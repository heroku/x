package signals

// Config describes the configurable parameters for Signals server.
type Config struct {
	ServerCloseWaitTime int `env:"SERVER_CLOSE_WAIT_TIME,default=0"`
}
