package debug

// Config describes the configurable parameters for debugging.
type Config struct {
	Port int `env:"DEBUG_PORT,default=9999"`
	PProfConfig
}

type PProfConfig struct {
	PProfPort            int  `env:"DEBUG_PPROF_PORT,default=9998"`
	EnablePProfDebugging bool `env:"DEBUG_PPROF_ENABLE,default=false"`

	// Mutex Profiling is not enabled by default
	EnableMutexProfiling bool `env:"DEBUG_PPROF_MUTEX_PROFILE_ENABLE,default=false"`

	// This controls how much of fraction of mutexes we need to consider for profiling.
	MutexProfileFraction int `env:"DBEUG_PPROF_MUTEX_PROFILE_FRACTION,default=2"`

	// Heap Profiling is enabled by default.
	// This controls the frequency of heap allocation sampling.
	// It defines the number of bytes allocated between samples.
	// Default is actual default value of MemProfileRate
	MemProfileRate int `env:"DBEUG_PPROF_MEM_PROFILE_RATE,default=524288"`

	// Block Profiling is not enabled default
	EnableBlockProfiling bool `env:"DEBUG_PPROF_BLOCK_PROFILE_ENABLE,default=false"`

	// This controls how much blocking time in nano seconds we need to consider.
	// Default 10 indicates, we consider all 10 ns blocking events .
	BlockProfileRate int `env:"DBEUG_PPROF_BLOCK_PROFILE_RATE,default=10"`
}
