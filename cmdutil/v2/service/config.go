package service

import (
	"time"

	"github.com/heroku/x/cmdutil/v2/debug"
	"github.com/heroku/x/cmdutil/v2/metrics"
	"github.com/heroku/x/cmdutil/v2/rollbar"
	"github.com/heroku/x/cmdutil/v2/svclog"
)

// standardConfig is used when service.New is called.
type standardConfig struct {
	Debug   debug.Config
	Logger  svclog.Config
	Metrics metrics.Config
	Rollbar rollbar.Config
}

// platformConfig is used by HTTP and captures
// config related to running on the Heroku platform.
type platformConfig struct {
	// Port is the primary port to listen on when running as a normal platform
	// app.
	Port int `env:"PORT"`

	// AdditionalPort defines an additional port to listen on in addition to the
	// primary port for use with dyno-dyno networking.
	AdditionalPort int `env:"ADDITIONAL_PORT"`
}

type timeoutConfig struct {
	Read       time.Duration `env:"SERVER_READ_TIMEOUT"`
	ReadHeader time.Duration `env:"SERVER_READ_HEADER_TIMEOUT,default=30s"`
	Write      time.Duration `env:"SERVER_WRITE_TIMEOUT"`
	Idle       time.Duration `env:"SERVER_IDLE_TIMEOUT"`
}
