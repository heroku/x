package debug

// Config describes the configurable parameters for debugging.
type Config struct {
	Port int `env:"DEBUG_PORT,default=9999"`
}
