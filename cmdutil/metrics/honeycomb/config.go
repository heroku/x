package honeycomb

// Config is a reusable configuration struct that contains the necessary
// environment variables to setup an OTEL provider with honeycomb.io.
type Config struct {
	MetricsDataset string `env:"HONEYCOMB_METRICS_DATASET"`
}
