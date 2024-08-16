package debug

// Config describes the configurable parameters for debugging.
type Config struct {
	Port                 int  `env:"DEBUG_PORT,default=9999"`
	PprofPort            int  `env:"PPROF_DEBUG_PORT,default=9998"`
	EnablePprofDebugging bool `env:"ENABLE_PPROF_DEBUGGING,default=false"`
	MutexProfileFraction int  `env:"MUTEX_PROFILE_FRACTION,default=2"`
}
