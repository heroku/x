// Package oc provides a cmdutil.Server for opencensus.
package oc

import (
	"context"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/trace"

	"github.com/heroku/x/cmdutil"
)

// Config of Open Census, via github.com/joeshaw/envdecode
//
// TODO[freeformz]: Support the other ocagent.WithXX options
type Config struct {
	// AgentAddress in the form of 'host:port'. Leave empty to disable.
	// (ocagent.WithAddress).
	AgentAddress string `env:"OC_AGENT_ADDR"`
	// ReconnectionPeriod to use when reconnecting to the agent. Defaults to
	// 5s.(ocagent.WithReconnectionPeriod).
	ReconnectionPeriod time.Duration `env:"OC_RECONNECTION_PERIOD,default=5s"`
	// WithInsecure. Defaults to false (ocagent.WithInsecure).
	WithInsecure bool `env:"OC_INSECURE,default=false"`
}

// ExporterOptions derived from the configuration.
func (c Config) ExporterOptions(serviceName string) []ocagent.ExporterOption {
	opts := []ocagent.ExporterOption{
		ocagent.WithAddress(c.AgentAddress),
		ocagent.WithReconnectionPeriod(c.ReconnectionPeriod),
		ocagent.WithServiceName(serviceName),
	}
	if c.WithInsecure {
		opts = append(opts, ocagent.WithInsecure())
	}
	return opts
}

// TraceConfig derived from the Config. This currently defaults to AlwaysSample
// and otherwise Default Max for the other trace.Config items.
//
// TODO[freeformz]: implement determining trace.Config via environment.
func (c Config) TraceConfig() trace.Config {
	return trace.Config{
		DefaultSampler:             trace.AlwaysSample(),
		MaxAttributesPerSpan:       trace.DefaultMaxAttributesPerSpan,
		MaxAnnotationEventsPerSpan: trace.DefaultMaxAnnotationEventsPerSpan,
		MaxMessageEventsPerSpan:    trace.DefaultMaxMessageEventsPerSpan,
		MaxLinksPerSpan:            trace.DefaultMaxLinksPerSpan,
	}
}

// NewExporter creates and registers an open census trace exporter as
// a cmdutil.Server wit the provided trace.Config / ocagent.ExporterOptions.
func NewExporter(tc trace.Config, opts ...ocagent.ExporterOption) (cmdutil.Server, error) {
	oce, err := ocagent.NewUnstartedExporter(opts...)
	if err != nil {
		return nil, err
	}
	trace.RegisterExporter(oce)
	trace.ApplyConfig(tc)

	return cmdutil.NewContextServer(
		func(ctx context.Context) error {
			if err := oce.Start(); err != nil {
				return err
			}
			<-ctx.Done() // wait for ctx to be canceled
			return oce.Stop()
		},
	), nil
}
