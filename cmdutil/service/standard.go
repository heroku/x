package service

import (
	"math/rand"
	"strings"
	"syscall"
	"time"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/debug"
	"github.com/heroku/x/cmdutil/metrics"
	"github.com/heroku/x/cmdutil/rollbar"
	"github.com/heroku/x/cmdutil/signals"
	"github.com/heroku/x/cmdutil/svclog"
	xmetrics "github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/l2met"
	"github.com/joeshaw/envdecode"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
)

func init() {
	// Initialize the pseudo-random number generator with a unique value so we
	// get unique sequences across runs.
	rand.Seed(time.Now().UnixNano())
}

// Standard is a standard service.
type Standard struct {
	g run.Group

	App             string
	Deploy          string
	Logger          logrus.FieldLogger
	MetricsProvider xmetrics.Provider
}

// New returns a Standard service with logging, rollbar, metrics, debugging,
// and common signal handling.
//
// It calls envdecode.MustStrictDecode on the provided appConfig.
func New(appConfig interface{}, ofs ...OptionFunc) *Standard {
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
	skipMetricsSuffix   bool
	customMetricsSuffix string
}

// OptionFunc is a function that modifies internal service options.
type OptionFunc func(*options)

// SkipMetricsSuffix is an OptionFunc that has New skip automatically
// adding the process type from DYNO as a suffix on metrics recorded with
// MetricsProvider.
func SkipMetricsSuffix() OptionFunc {
	return func(o *options) {
		o.skipMetricsSuffix = true
	}
}

// CustomMetricsSuffix is an OptionFunc that has New use the given suffix
// on metrics recorded with MetricsProvider instead of inferring it from
// DYNO.
func CustomMetricsSuffix(s string) OptionFunc {
	return func(o *options) {
		o.customMetricsSuffix = s
	}
}

// metricsSuffixFromDyno determines a metrics suffix from
// dyno. It uses the process type component from dyno, or
// "server" if that's "web."
// If dyno is empty, it returns an empty suffix.
func metricsSuffixFromDyno(dyno string) string {
	if dyno == "" {
		return ""
	}
	parts := strings.SplitN(dyno, ".", 2)
	pt := parts[0]
	if pt == "web" {
		pt = "server"
	}
	return pt
}
