package lambda

import (
	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/joeshaw/envdecode"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/metrics"
	"github.com/heroku/x/cmdutil/rollbar"
	"github.com/heroku/x/cmdutil/svclog"
	xmetrics "github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/l2met"
)

// Function defines configuration of a Lambda function.
type Function struct {
	// Name of the function. This will be equivalent to the APP_NAME env var.
	Name string
	// Deploy is a cloud identifier.
	Deploy string
	// Logger is a field logger.
	Logger logrus.FieldLogger
	// Metrics provider defines interactions for recording metrics.
	MetricsProvider xmetrics.Provider

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

	rollbar.Setup(logger, fc.Rollbar)

	f := &Function{
		Name:   fc.Logger.AppName,
		Deploy: fc.Logger.Deploy,
		Logger: logger,
	}

	if fc.Metrics.Librato.User != "" {
		f.MetricsProvider = metrics.StartLibrato(logger, fc.Metrics)
	} else {
		l2met := l2met.New(logger)
		f.MetricsProvider = l2met
		f.Add(cmdutil.NewContextServer(l2met.Run))
	}

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
		defer rollbar.ReportPanic(f.Logger)

		// Run any background servers, if configured.
		// For example, the l2met agent.
		err := f.g.Run()

		if err != nil {
			f.Logger.WithError(err).Error("background server ended in error")
		}
	}()

	awslambda.Start(handler)
}

type funcConfig struct {
	Logger  svclog.Config
	Metrics metrics.Config
	Rollbar rollbar.Config
}
