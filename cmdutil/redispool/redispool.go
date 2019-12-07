// Package redispool supports setting up redis connection pools parameterized
// via environment variables.
package redispool

import (
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

// Config stores all the basic redis pool configuration knobs.
type Config struct {
	// URL is the redis URL the pool connects to, configured via `REDIS_URL`.
	URL string `env:"REDIS_URL,default=redis://127.0.0.1:6379"`

	// MaxIdleConns is the maximum number of idle connections in a pool,
	// configured via `REDIS_MAX_IDLE_CONNS`, defaults to 1.
	MaxIdleConns int `env:"REDIS_MAX_IDLE_CONNS,default=1"`

	// MaxActiveConns is the maximum number of active connections in a pool,
	// configured via `REDIS_MAX_ACTIVE_CONNS`, defaults to 50.
	MaxActiveConns int `env:"REDIS_MAX_ACTIVE_CONNS,default=50"`

	// MaxConnLifetime is the maximum lifetime of any connection in a pool,
	// configured via `REDIS_MAX_CONN_LIFETIME`, defaults to 0s.
	// Connections older than this duration will be closed. If the value is zero,
	// then the pool does not close connections based on age.
	MaxConnLifetime time.Duration `env:"REDIS_MAX_CONN_LIFETIME,default=0s"`

	// IdleTimeout is the duration how long connections can be idle before they're
	// closed, configured via `REDIS_IDLE_TIMEOUT`. When set to zero idle
	// connections won't be closed. Defaults to 0s.
	IdleTimeout time.Duration `env:"REDIS_IDLE_TIMEOUT,default=0s"`

	// IdleProbeInterval is the duration between the aliveness checks of idle
	// connections, configured via `REDIS_IDLE_PROBE_INTERVAL` which defaults to 0.
	// When set to zero, aliveness is not checked.
	IdleProbeInterval time.Duration `env:"REDIS_IDLE_PROBE_INTERVAL,default=0s"`

	// AttachmentNames is an optional configuration which allows to set up
	// connection pools to each redis attachment using the Pools() function. The
	// same configuration parameters apply for each pool, except the URL which is
	// extracted from the environment based on the attachment name.
	AttachmentNames []string `env:"REDIS_ATTACHMENT_NAMES"`
}

// Pool returns a *redis.Pool given the configured env vars in Config.
func (c Config) Pool() *redis.Pool {
	return newPool(c)
}

// Pools returns a *redis.Pool for each of the provided Config values.
func (c Config) Pools() ([]*redis.Pool, error) {
	pools := make([]*redis.Pool, len(c.AttachmentNames))

	for i, attachment := range c.AttachmentNames {
		base := c
		base.URL = os.Getenv(attachment + "_URL")
		if base.URL == "" {
			return nil, errors.Errorf("missing redis attachment url for %v", attachment)
		}
		pools[i] = newPool(base)
	}

	return pools, nil
}

func newPool(cfg Config) *redis.Pool {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			conn, err := redis.DialURL(cfg.URL)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
		MaxIdle:         cfg.MaxIdleConns,
		MaxActive:       cfg.MaxActiveConns,
		MaxConnLifetime: cfg.MaxConnLifetime,
		IdleTimeout:     cfg.IdleTimeout,
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			var err error
			if cfg.IdleProbeInterval > 0 && time.Since(t) > cfg.IdleProbeInterval {
				_, err = conn.Do("PING")
			}
			return err
		},
	}
}
