package https

import "time"

// Config for HTTP and HTTPS servers.
type Config struct {
	InsecurePort int `env:"HEROKU_ROUTER_HTTP_PORT,required"`
	SecurePort   int `env:"HEROKU_ROUTER_HTTPS_PORT,required"`

	// These environment variables are automatically set by ACM in
	// relation to Let's Encrypt certificates.
	ServerCert  string `env:"SERVER_CERT,required"`
	ServerKey   string `env:"SERVER_KEY,required"`
	UseAutocert bool   `env:"HTTPS_USE_AUTOCERT"`
	// These environment variables are set with default values of 60s
	ReadTimeout  time.Duration `env:"HTTP_SERVER_READ_TIMEOUT,default=60s"`
	WriteTimeout time.Duration `env:"HTTP_SERVER_WRITE_TIMEOUT,default=60s"`
}
