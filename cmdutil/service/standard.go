package service

import (
	"math/rand"
	"strings"
	"syscall"
	"time"

	"github.com/joeshaw/envdecode"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/debug"
	"github.com/heroku/x/cmdutil/metrics"
	"github.com/heroku/x/cmdutil/oc"
	"github.com/heroku/x/cmdutil/rollbar"
	"github.com/heroku/x/cmdutil/signals"
	"github.com/heroku/x/cmdutil/svclog"
	xmetrics "github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/l2met"
)

// Standard is a standard service.
type Standard struct {
	g run.Group

	App             string
	Deploy          string
	Logger          logrus.FieldLogger
	MetricsProvider xmetrics.Provider
}

// New Standard Service with logging, rollbar, metrics, debugging, common signal
// handling, and possibly more. envdecode.MustStrictDecode is called on the
// provided appConfig to ensure that it is processed.
func New(appConfig interface{}, ofs ...OptionFunc) *Standard {
	// Initialize the pseudo-random number generator with a unique value so we
	// get unique sequences across runs.
	rand.Seed(time.Now().UnixNano())

	var sc standardConfig
	envdecode.MustStrictDecode(&sc)
	envdecode.MustStrictDecode(appConfig)

	logger := svclog.NewLogger(sc.Logger)

	rollbar.Setup(logger, sc.Rollbar)

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

	if sc.Metrics.Librato.User != "" {
		s.MetricsProvider = metrics.StartLibrato(logger, sc.Metrics)
	} else {
		l2met := l2met.New(logger)
		s.MetricsProvider = l2met
		s.Add(cmdutil.NewContextServer(l2met.Run))
	}

	s.Add(debug.New(logger, sc.Debug.Port))
	s.Add(signals.NewServer(logger, syscall.SIGINT, syscall.SIGTERM))

	// only setup an exporter if indicated && the AgentAddress is set
	// this separates the code change saying yes, do tracing from
	// the operational aspect of deciding where it goes.
	if o.enableOpenCensusTracing && sc.OpenCensus.AgentAddress != "" {
		oce, err := oc.NewExporter(
			sc.OpenCensus.TraceConfig(),
			sc.OpenCensus.ExporterOptions(s.App)...,
		)
		if err != nil {
			panic(err)
		}
		s.Add(oce)
	}

	return s
}

// Add adds cmdutil.Servers to be managed.
func (s *Standard) Add(svs ...cmdutil.Server) {
	for _, sv := range svs {
		s.g.Add(sv.Run, sv.Stop)
	}
}

// Run runs all standard and Added cmdutil.Servers.
//
// If a panic is encountered, it is reported to Rollbar.
//
// If the error returned by oklog/run.Run is non-nil, it is logged
// with s.Logger.Fatal.
func (s *Standard) Run() {
	defer rollbar.ReportPanic(s.Logger)

	err := s.g.Run()

	// Not using defer here since it will have no effect if Fatal below
	// is called.
	s.MetricsProvider.Stop()

	if err != nil {
		s.Logger.WithError(err).Fatal()
	}
}

type options struct {
	customMetricsSuffix     string
	skipMetricsSuffix       bool
	enableOpenCensusTracing bool
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

// EnableOpenCensusTracing on the Service by registering and starting an open
// census agent exporter.
func EnableOpenCensusTracing() OptionFunc {
	return func(o *options) {
		o.enableOpenCensusTracing = true
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
