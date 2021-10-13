package otel

import (
	"encoding/base64"
	"net/url"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"

	"github.com/heroku/x/tlsconfig"
)

// NewHTTPClient creates a new HTTP client for exporting otel metrics.
func NewHTTPClient(url *url.URL, opts ...otlpmetrichttp.Option) otlpmetric.Client {
	userInfo := url.User
	authHeader := make(map[string]string)
	authHeader["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(userInfo.String()))

	// Ensure there's no cred in the URL.
	url.User = nil

	defaultOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(url.Hostname()),
		otlpmetrichttp.WithTLSClientConfig(tlsconfig.New()),
		otlpmetrichttp.WithHeaders(authHeader),
	}

	defaultOpts = append(defaultOpts, opts...)

	return otlpmetrichttp.NewClient(defaultOpts...)
}
