// Package metrics provides helpers for setting up metrics reporting.
package metrics

import (
	"net/url"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/provider/librato"
	"github.com/heroku/x/go-kit/runtimemetrics"
)

// StartLibrato initializes a new librato provider given the Config, and sets
// up a runtimemetrics.Collector on it. The runtime collector will emit on
// the librato.Provider every Config.ReportInterval ticks. You need to call
// Stop on the librato.Provider before you tear down the process.
func StartLibrato(logger log.FieldLogger, cfg Config) metrics.Provider {
	logger = logger.WithField("at", "librato")

	l := librato.New(
		cfg.Librato.URL(),
		cfg.ReportInterval,
		librato.WithSource(cfg.Source),
		librato.WithPrefix(cfg.Prefix),
		librato.WithPercentilePrefix(".perc"),
		librato.WithResetCounters(),
		librato.WithSSA(),
		librato.WithErrorHandler(func(err error) {
			logLibratoError(logger, err)
		}),
		librato.WithRequestDebugging(),
	)

	if cfg.Librato.TagsEnabled {
		librato.WithTags(cfg.DefaultTags...)(l.(*librato.Provider))
	}

	c := runtimemetrics.NewCollector(l)
	go func() {
		for range time.Tick(cfg.ReportInterval) {
			c.Collect()
		}
	}()

	return l
}

// Config stores all the env related config to bootstrap metrics.
type Config struct {
	ReportInterval time.Duration `env:"METRICS_REPORT_INTERVAL,default=60s"`
	Source         string        `env:"METRICS_SOURCE"`
	Prefix         string        `env:"METRICS_PREFIX"`
	DefaultTags    []string      `env:"METRICS_DEFAULT_TAGS"`
	Librato        Librato
}

// Librato stores all related librato config to be able to connect to its API.
type Librato struct {
	APIURL      *url.URL `env:"LIBRATO_API_URL"`
	User        string   `env:"LIBRATO_USER"`
	Password    string   `env:"LIBRATO_PASSWORD"`
	TagsEnabled bool     `env:"LIBRATO_TAGS_ENABLED"`
}

// URL returns the specified LIBRATO_API_URL, if any. Otherwise it defaults to
// the default Librato URL. The credentials are applied from LIBRATO_USER and
// LIBRATO_PASSWORD.
func (l Librato) URL() *url.URL {
	if l.APIURL != nil {
		l.APIURL.User = url.UserPassword(l.User, l.Password)
		return l.APIURL
	}

	u, err := url.Parse(librato.DefaultURL)
	if err != nil {
		panic(errors.Wrap(err, "librato URL invalid"))
	}
	u.User = url.UserPassword(l.User, l.Password)
	return u
}

type requester interface {
	Request() string
}

func logLibratoError(l log.FieldLogger, err error) {
	if r, ok := err.(requester); ok {
		l = l.WithField("request_body", r.Request())
	}
	l.WithError(err).Warn()
}
