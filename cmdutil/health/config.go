package health

// Config can be used in a service's main config struct to load the
// healthcheck port from the environment.
type Config struct {
	Port           int `env:"HEROKU_ROUTER_HEALTHCHECK_PORT,default=6000"`
	MetricInterval int `env:"HEROKU_HEALTH_METRIC_INTERVAL,default=5"`
}
