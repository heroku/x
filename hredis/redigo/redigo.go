package redigo

import (
	"net/url"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
)

// WaitFunc to be executed occasionally by something that is waiting.
// Should return an error to cancel the waiting
// Should also sleep some amount of time to throttle connection attempts
type WaitFunc func(time.Time) error

// WaitForAvailability of the redis server located at the provided url, timeout if the Duration passes before being able to connect
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

// DialURLFunc describes the type of redis.DialURL, useful for changing the behavior
// of a redis Dialer.
type DialURLFunc func(string, ...redis.DialOption) (redis.Conn, error)

// RedisDialer is an internal helper providing custom Redis dial logic for
// authentication purposes.
type RedisDialer struct {
	// Stripped of authentication, but in the form redis://host:port, as in redis.DialURL
	urlToDial string
	passwords []string
	dialURL   DialURLFunc

	// most likely to be successful password. This may change throughout a Pool's lifecycle.
	mu       sync.Mutex
	password string
}

func (r *RedisDialer) addPassword(pass string) {
	r.passwords = append(r.passwords, pass)
}

// gets most likely to succeed password.
func (r *RedisDialer) getPassword() string {
	r.mu.Lock()
	tmp := r.password
	r.mu.Unlock()
	return tmp
}

// safely set's the most likely to succeed password.
func (r *RedisDialer) setPassword(pass string) {
	r.mu.Lock()
	r.password = pass
	r.mu.Unlock()
}

func (r *RedisDialer) dial() (redis.Conn, error) {
	c, err := r.dialURL(r.urlToDial)
	if err != nil {
		return nil, err
	}

	if len(r.passwords) == 0 {
		return c, nil
	}

	var authErr error

	lastPass := r.getPassword()

	if _, authErr = c.Do("AUTH", lastPass); authErr != nil {
	search:
		for _, pass := range r.passwords {
			if pass == lastPass {
				continue
			}

			if _, authErr = c.Do("AUTH", pass); authErr == nil {
				// nominate this pass as valid.
				r.setPassword(pass)
				break search
			}
		}
	}

	if authErr != nil {
		c.Close()
		return nil, authErr
	}

	return c, nil
}

type OptionFunc func(*RedisDialer)

func WithPasswords(passes ...string) OptionFunc {
	return func(dialer *RedisDialer) {
		dialer.passwords = passes
	}
}

func WithDialURLFunc(d DialURLFunc) OptionFunc {
	return func(dialer *RedisDialer) {
		dialer.dialURL = d
	}
}

// NewRedisPoolFromURL returns a new *redigo/redis.Pool configured for the supplied url
// The url can include a password in the standard form and if so is used to AUTH against
// the redis server
func NewRedisPoolFromURL(rawURL string, opts ...OptionFunc) (*redis.Pool, error) {
	// Extract / remove password from URL string
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	pass, _ := u.User.Password()
	u.User = url.UserPassword("", "")

	dialer := &RedisDialer{
		dialURL:   redis.DialURL,
		urlToDial: u.String(),
	}

	// opt may be a WithPasswords. The original password from the URL needs to be added
	// after options are run.
	for _, opt := range opts {
		opt(dialer)
	}

	if len(pass) > 0 {
		dialer.addPassword(pass)
		dialer.setPassword(pass)
	}

	// DialURL will error if wrong password is set. Not providing a password
	// to DialURL results in the ability to try all potential passwords before
	// erroring in our own Dial function.

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
