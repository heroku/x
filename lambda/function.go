package lambda

import (
	"context"

	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/joeshaw/envdecode"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/metrics"
	"github.com/heroku/x/cmdutil/metrics/otel"
	"github.com/heroku/x/cmdutil/rollbar"
	"github.com/heroku/x/cmdutil/svclog"
	kitmetrics "github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/l2met"
	"github.com/heroku/x/go-kit/metrics/multiprovider"
	xotel "github.com/heroku/x/go-kit/metrics/provider/otel"
)

const (
	// This is the key for the "function" OTEL attribute. The value should be the function name.
	functionKey = "function"

	// This is the key for the "stage" log field. The value should be "staging" or "production".
	stageKey = "stage"
)

// Function defines configuration of a Lambda function.
type Function struct {
	// App to which this function belongs. This will be equivalent to the APP_NAME env var.
	App string
	// Name of the function. This will be equivalent to the FUNCTION_NAME env var.
	Name string
	// Deploy is a cloud identifier.
	Deploy string
	// Stage is an env identifier (e.g. "staging" or "production").
	Stage string
	// Logger is a field logger.
	Logger logrus.FieldLogger
	// Metrics provider defines interactions for recording metrics.
	MetricsProvider kitmetrics.Provider

	g run.Group
}

// New creates a new Function with a configured logger, rollbar agent and metrics provider.
func New(config interface{}) *Function {
	var fc funcConfig
	envdecode.MustStrictDecode(&fc)

	if config != nil {
		envdecode.MustStrictDecode(config)
	}

	logger := svclog.NewLogger(fc.Logger)
	logger = logger.WithFields(logrus.Fields{
		functionKey: fc.Name,
		stageKey:    fc.Stage,
	})

	rollbar.Setup(logger, fc.Rollbar)

	f := &Function{
		App:    fc.Logger.AppName,
		Name:   fc.Name,
		Deploy: fc.Logger.Deploy,
		Stage:  fc.Stage,
		Logger: logger,
	}

	metricsProviders := []kitmetrics.Provider{}

	if fc.Metrics.OTEL.CollectorURL != nil && fc.Metrics.OTEL.Enabled {
		otelProvider := otel.MustProvider(
			context.Background(),
			logger,
			fc.Metrics.OTEL,
			f.App,
			f.Deploy,
			f.Stage,
			"lambda",
			xotel.WithAttributes(attribute.String(functionKey, f.Name)),
		)
		metricsProviders = append(metricsProviders, otelProvider)
	}

	if len(metricsProviders) == 0 {
		// Fallback to l2met when none configured.
		l2met := l2met.New(logger)
		metricsProviders = append(metricsProviders, l2met)
		f.Add(cmdutil.NewContextServer(l2met.Run))
	}

	f.MetricsProvider = multiprovider.New(metricsProviders...)

	return f
}

// Add adds cmdutil.Servers to be run in the background.
func (f *Function) Add(svs ...cmdutil.Server) {
	for _, sv := range svs {
		f.g.Add(sv.Run, sv.Stop)
	}
}

/* Start takes a lambda handler that must satisfy one of the following signatures:
 * 	func ()
 * 	func () error
 * 	func (TIn) error
 * 	func () (TOut, error)
 * 	func (TIn) (TOut, error)
 * 	func (context.Context) error
 * 	func (context.Context, TIn) error
 * 	func (context.Context) (TOut, error)
 * 	func (context.Context, TIn) (TOut, error)
 *
 * Where "TIn" and "TOut" are types compatible with the "encoding/json" standard library.
 * See https://golang.org/pkg/encoding/json/#Unmarshal for how deserialization behaves.
 *
 * See https://github.com/aws/aws-lambda-go/blob/main/lambda/entry.go for more info.
 */
func (f *Function) Start(handler interface{}) {
	defer f.MetricsProvider.Stop()

	// Run logger, rollbar agent and metrics provider in the background.
	go func() {
		defer svclog.ReportPanic(f.Logger)

		// Run any background servers, if configured.
		// For example, the l2met agent.
		err := f.g.Run()

		if err != nil {
			f.Logger.WithError(err).Error("background server ended in error")
		}
	}()

	awslambda.Start(handler)
}

// FlushMetrics is a hook for flushing metrics,
// to ensure they are sent before the invocation context is torn down.
func (f *Function) FlushMetrics() error {
	return f.MetricsProvider.Flush()
}

type funcConfig struct {
	Name    string `env:"FUNCTION_NAME"`
	Stage   string `env:"STAGE"`
	Logger  svclog.Config
	Metrics metrics.Config
	Rollbar rollbar.Config
}
