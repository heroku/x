package redispool

import (
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

// Config stores all the basic redis pool configuration knobs.
type Config struct {
	URL               string        `env:"REDIS_URL,default=redis://127.0.0.1:6379"`
	MaxIdleConns      int           `env:"REDIS_MAX_IDLE_CONNS,default=1"`
	MaxActiveConns    int           `env:"REDIS_MAX_ACTIVE_CONNS,default=50"`
	IdleTimeout       time.Duration `env:"REDIS_IDLE_TIMEOUT,default=0s"`
	IdleProbeInterval time.Duration `env:"REDIS_IDLE_PROBE_INTERVAL,default=0s"`
	AttachmentNames   []string      `env:"REDIS_ATTACHMENT_NAMES"`
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
		MaxIdle:     cfg.MaxIdleConns,
		MaxActive:   cfg.MaxActiveConns,
		IdleTimeout: cfg.IdleTimeout,
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			var err error
			if cfg.IdleProbeInterval > 0 && time.Since(t) > cfg.IdleProbeInterval {
				_, err = conn.Do("PING")
			}
			return err
		},
	}
}
