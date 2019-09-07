package redigo

import (
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
)

// WaitFunc to be executed occasionally by something that is waiting.
// Should return an error to cancel the waiting
// Should also sleep some amount of time to throttle connection attempts
type WaitFunc func(time.Time) error

// WaitForAvailability of the redis server located at the provided url, timeout
// if the Duration passes before being able to connect
func WaitForAvailability(url string, d time.Duration, f WaitFunc) (bool, error) {
	conn := make(chan struct{})
	errs := make(chan error)
	go func() {
		for {
			c, err := redis.DialURL(url)
			if err == nil {
				c.Close()
				close(conn)
				return
			}
			if f != nil {
				err := f(time.Now())
				if err != nil {
					errs <- err
					return
				}
			}
		}
	}()

	select {
	case err := <-errs:
		return false, err
	case <-conn:
		return true, nil
	case <-time.After(d):
		return false, nil
	}
}

// redisDialer is a helper that provides custom Redis dial logic for
// authentication purposes.
type redisDialer struct {
	url       string      // Stripped of authentication, but in the form redis://host:port, as in redis.DialURL
	passwords []string    // The passwords that will be tried for authentication
	dialURL   DialURLFunc // Defaults to redis.DialURL
}

func (d *redisDialer) dial() (redis.Conn, error) {
	c, err := d.dialURL(d.url)
	if err != nil || len(d.passwords) == 0 {
		// error or no passwords to try
		return c, err
	}

	for _, pass := range d.passwords {
		if _, err = c.Do("AUTH", pass); err == nil {
			return c, nil
		}
	}

	// Went through all the passwords, last one still errored, so no passwords
	// work, close the connection to prevent a leak.
	if err != nil {
		c.Close()
	}

	return c, err
}

// OptionFunc sets redisDialer options
type OptionFunc func(*redisDialer)

// WithPasswords specifies additional passwords to try authenticating with when
// dialing
func WithPasswords(passes ...string) OptionFunc {
	return func(d *redisDialer) {
		d.passwords = append(d.passwords, passes...)
	}
}

// DialURLFunc describes the type implemented by redis.DialURL, useful for
// changing the behavior of a redis Dialer.
type DialURLFunc func(string, ...redis.DialOption) (redis.Conn, error)

// WithDialURLFunc specifies an alternative DialURLFunc to use when dialing via
// URL.
func WithDialURLFunc(df DialURLFunc) OptionFunc {
	return func(d *redisDialer) {
		d.dialURL = df
	}
}

// NewRedisPoolFromURL returns a new *redigo/redis.Pool configured for the
// supplied url. If the url includes a password in the standard form it is used
// to AUTH against the redis server
func NewRedisPoolFromURL(rawURL string, opts ...OptionFunc) (*redis.Pool, error) {
	// Extract / remove password from URL string
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	var pwds []string
	pass, ok := u.User.Password()
	if ok {
		pwds = append(pwds, pass)
		u.User = url.UserPassword("", "")
	}

	dialer := &redisDialer{
		url:       u.String(),
		dialURL:   redis.DialURL,
		passwords: pwds,
	}

	for _, opt := range opts {
		opt(dialer)
	}

	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial:        dialer.dial,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}, nil
}
