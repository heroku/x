package service

import (
	"net/url"

	"github.com/heroku/x/cmdutil/debug"
	"github.com/heroku/x/cmdutil/metrics"
	"github.com/heroku/x/cmdutil/oc"
	"github.com/heroku/x/cmdutil/rollbar"
	"github.com/heroku/x/cmdutil/svclog"
)

// standardConfig is used when service.New is called.
type standardConfig struct {
	Debug      debug.Config
	Logger     svclog.Config
	Metrics    metrics.Config
	Rollbar    rollbar.Config
	OpenCensus oc.Config
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

// bypassConfig is used by HTTP and GRPC and captures
// config related to running with the router bypass
// feature on the Heroku platform.
type bypassConfig struct {
	// The following ports, TLS, and ACM configurations are set when running with
	// spaces-router-bypass enabled.
	InsecurePort          int `env:"HEROKU_ROUTER_HTTP_PORT"`
	SecurePort            int `env:"HEROKU_ROUTER_HTTPS_PORT"`
	HealthPort            int `env:"HEROKU_ROUTER_HEALTHCHECK_PORT"`
	TLS                   tlsConfig
	ACMEHTTPValidationURL *url.URL `env:"ACME_HTTP_VALIDATION_URL,default=https://va-acm.runtime.herokai.com/challenge"`
}

// tlsConfig is used by bypassConfig and captures config related to TLS
// when not running on the Heroku platform.
type tlsConfig struct {
	// These environement variables are automatically set by Foundation in
	// relation to Let's Encrypt certificates.
	ServerCert string `env:"SERVER_CERT"`
	ServerKey  string `env:"SERVER_KEY"`

	// Used by GRPC services, set by terraform.
	ServerCACert string `env:"SERVER_CA_CERT"`

	UseAutocert bool `env:"HTTPS_USE_AUTOCERT"`
}

// spaceCAConfig is used by grpcConfig and captures config related to
// common runtime services whose certs are generated using the
// spaceCA.
type spaceCAConfig struct {
	// Used by GRPC services in new mTLS cert generation where services
	// generate their certificates using the SpaceCA.
	RootCACert  string `env:"HEROKU_SPACE_CA_ROOT_CERT"`
	SpaceCACert string `env:"HEROKU_SPACE_CA_CERT"`
	SpaceCAKey  string `env:"HEROKU_SPACE_CA_KEY"`

	// RootCACertAlternate is set during a root certificate rotation and must be
	// installed into the certificate pool to ensure that services are able to
	// communicate while the rotation is in progress.
	RootCACertAlternate string `env:"HEROKU_SPACE_CA_ROOT_CERT_ALTERNATE"`

	// Switch which will determine whether an app generates their cert
	// using the SpaceCA.
	UseSpaceCA bool `env:"USE_SPACE_CA,default=false"`

	// Domain of the service used in the generation of the cert
	Domain string `env:"DOMAIN"`
}
