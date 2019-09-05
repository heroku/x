package https

// Config for HTTP and HTTPS servers.
type Config struct {
	InsecurePort int `env:"HEROKU_ROUTER_HTTP_PORT,required"`
	SecurePort   int `env:"HEROKU_ROUTER_HTTPS_PORT,required"`

	// These environement variables are automatically set by ACM in
	// relation to Let's Encrypt certificates.
	ServerCert  string `env:"SERVER_CERT,required"`
	ServerKey   string `env:"SERVER_KEY,required"`
	UseAutocert bool   `env:"HTTPS_USE_AUTOCERT"`
}
