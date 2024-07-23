package service

import (
	"os"
	"strings"
	"syscall"

	"log/slog"

	"github.com/joeshaw/envdecode"
	"github.com/oklog/run"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/metrics"
	"github.com/heroku/x/cmdutil/v2/debug"
	"github.com/heroku/x/cmdutil/v2/signals"
	"github.com/heroku/x/cmdutil/v2/svclog"
	xmetrics "github.com/heroku/x/go-kit/metrics"
)

// Standard is a standard service.
type Standard struct {
	g run.Group

	App             string
	Deploy          string
	Logger          *slog.Logger
	MetricsProvider xmetrics.Provider
}

// New Standard Service with logging, rollbar, metrics, debugging, common signal
// handling, and possibly more.
//
// If appConfig is non-nil, envdecode.MustStrictDecode will be called on it
// to ensure that it is processed.
func New(appConfig interface{}, ofs ...OptionFunc) *Standard {
	var sc standardConfig
	envdecode.MustStrictDecode(&sc)

	if appConfig != nil {
		envdecode.MustStrictDecode(appConfig)
	}

	logger := svclog.NewLogger(sc.Logger)

	// TODO: Add rollbar support.
	var o options
	for _, of := range ofs {
		of(&o)
	}

	if !o.skipMetricsSuffix && sc.Metrics.Prefix != "" {
		suf := o.customMetricsSuffix
		if suf == "" {
			suf = metricsSuffixFromDyno(sc.Logger.Dyno)
		}
		if suf != "" {
			sc.Metrics.Prefix += "." + suf
		}
	}

	s := &Standard{
		App:    sc.Logger.AppName,
		Deploy: sc.Logger.Deploy,
		Logger: logger,
	}

	s.Add(debug.New(logger, sc.Debug.Port))
	s.Add(signals.NewServer(logger, syscall.SIGINT, syscall.SIGTERM))

	return s
}

// Add adds cmdutil.Servers to be managed.
func (s *Standard) Add(svs ...cmdutil.Server) {
	for _, sv := range svs {
		sv := sv
		runWithPanicReport := func() error {
			defer metrics.ReportPanic(s.MetricsProvider)
			defer svclog.ReportPanic(s.Logger)
			return sv.Run()
		}
		s.g.Add(runWithPanicReport, sv.Stop)
	}
}

// Run runs all standard and Added cmdutil.Servers.
//
// If a panic is encountered, it is reported to Rollbar.
//
// If the error returned by oklog/run.Run is non-nil, it is logged
// with s.Logger.Fatal.
func (s *Standard) Run() {
	err := s.g.Run()

	if s.MetricsProvider != nil {
		s.MetricsProvider.Stop()
	}

	if err != nil {
		s.Logger.Error(err.Error())
		os.Exit(1)
	}
}

type options struct {
	customMetricsSuffix string
	skipMetricsSuffix   bool
}

// OptionFunc is a function that modifies internal service options.
type OptionFunc func(*options)

// SkipMetricsSuffix prevents the Service from suffixing the process type to
// metric names recorded by the MetricsProvider. The default suffix is
// determined from $DYNO.
func SkipMetricsSuffix() OptionFunc {
	return func(o *options) {
		o.skipMetricsSuffix = true
	}
}

// CustomMetricsSuffix to be added to metrics recorded by the MetricsProvider
// instead of inferring it from $DYNO.
func CustomMetricsSuffix(s string) OptionFunc {
	return func(o *options) {
		o.customMetricsSuffix = s
	}
}

// metricsSuffixFromDyno determines a metrics suffix from the process part of
// $DYNO. If $DYNO indicates a "web" process, the suffix is "server". If $DYNO
// is empty, so is the suffix.
func metricsSuffixFromDyno(dyno string) string {
	if dyno == "" {
		return dyno
	}
	parts := strings.SplitN(dyno, ".", 2)
	if parts[0] == "web" {
		parts[0] = "server" // TODO[freeformz]: Document why this is server
	}
	return parts[0]
}
