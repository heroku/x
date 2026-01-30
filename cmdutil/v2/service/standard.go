package service

import (
	"context"
	"log/slog"
	"syscall"

	"github.com/joeshaw/envdecode"
	"github.com/oklog/run"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/heroku/x/cmdutil/v2"
	"github.com/heroku/x/cmdutil/v2/debug"
	"github.com/heroku/x/cmdutil/v2/metrics"
	"github.com/heroku/x/cmdutil/v2/signals"
	"github.com/heroku/x/cmdutil/v2/svclog"
)

// Standard is a standard service.
type Standard struct {
	g run.Group

	App             string
	Deploy          string
	Logger          *slog.Logger
	Meter           *metric.MeterProvider
	shutdownMetrics func(context.Context) error
}

// New Standard Service with logging, metrics, debugging, and signal handling.
//
// If appConfig is non-nil, envdecode.MustStrictDecode will be called on it.
func New(appConfig interface{}, ofs ...OptionFunc) *Standard {
	var sc standardConfig
	envdecode.MustStrictDecode(&sc)

	if appConfig != nil {
		envdecode.MustStrictDecode(appConfig)
	}

	logger := svclog.NewLogger(sc.Logger)

	var o options
	for _, of := range ofs {
		of(&o)
	}

	s := &Standard{
		App:    sc.Logger.AppName,
		Deploy: sc.Logger.Deploy,
		Logger: logger,
	}

	if sc.Metrics.Enabled {
		mp, shutdown, err := metrics.Setup(
			context.Background(),
			sc.Metrics,
			sc.Logger.AppName,
			"heroku",
			sc.Logger.Deploy,
			sc.Logger.Dyno,
		)
		if err != nil {
			logger.Error("failed to setup metrics", slog.Any("error", err))
			panic("failed to setup metrics: " + err.Error())
		}
		s.Meter = mp
		s.shutdownMetrics = shutdown
	}

	s.Add(debug.New(logger, sc.Debug))
	s.Add(signals.NewServer(logger, syscall.SIGINT, syscall.SIGTERM))

	return s
}

// Add adds cmdutil.Servers to be managed.
func (s *Standard) Add(svs ...cmdutil.Server) {
	for _, sv := range svs {
		runWithPanicReport := func() error {
			defer func() {
				if r := recover(); r != nil {
					s.Logger.Error("panic", "panic", r)
					panic(r)
				}
			}()
			return sv.Run()
		}
		s.g.Add(runWithPanicReport, sv.Stop)
	}
}

// Run runs all standard and Added cmdutil.Servers.
//
// If the error returned by oklog/run.Run is non-nil, it is logged.
func (s *Standard) Run() {
	err := s.g.Run()

	if s.shutdownMetrics != nil {
		if shutdownErr := s.shutdownMetrics(context.Background()); shutdownErr != nil {
			s.Logger.Error("failed to shutdown metrics", slog.Any("error", shutdownErr))
		}
	}

	if err != nil {
		s.Logger.Error("service error", slog.Any("error", err))
	}
}

type options struct{}

// OptionFunc is a function that modifies internal service options.
type OptionFunc func(*options)
