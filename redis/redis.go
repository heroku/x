package redis

import (
	"net/url"
	"time"

	"github.com/garyburd/redigo/redis"
)

// WaitForAvailability of the redis server located at the provided url, timeout if the Duration passes before being able to connect
func WaitForAvailability(url string, d time.Duration) (bool, error) {
	h, _, err := ParseURL(url)
	if err != nil {
		return false, err
	}
	conn := make(chan struct{})
	go func() {
		for {
			c, err := redis.Dial("tcp", h)
			if err == nil {
				c.Close()
				conn <- struct{}{}
				return
			}
			time.Sleep(d / 100)
		}
	}()
	select {
	case <-conn:
		return true, nil
	case <-time.After(d):
		return false, nil
	}
}

// ParseURL in the form of redis://h:<pwd>@ec2-23-23-129-214.compute-1.amazonaws.com:25219
// and return the host and password
func ParseURL(us string) (string, string, error) {
	u, err := url.Parse(us)
	if err != nil {
		return "", "", err
	}
	var password string
	if u.User != nil {
		password, _ = u.User.Password()
	}
	var host string
	if u.Host == "" {
		host = "localhost"
	} else {
		host = u.Host
	}
	return host, password, nil
}

// NewRedisPoolFromURL returns a new *redigo/redis.Pool configured for the supplied url
// The url can include a password in the standard form and if so is used to AUTH against
// the redis server
func NewRedisPoolFromURL(url string) (*redis.Pool, error) {
	h, p, err := ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", h)
			if err != nil {
				return nil, err
			}
			if p != "" {
				if _, err := c.Do("AUTH", p); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}, nil
}
