package signals

import "time"

// Config describes the configurable parameters for Signals server.
type Config struct {
	SignalsServerStopDelay time.Duration `env:"SIGNALS_SERVER_STOP_DELAY,default=0s"`
}
