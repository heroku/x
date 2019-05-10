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

// make this a var, so we can change this for testing.
var redisDialURL = redis.DialURL

// NewRedisPoolFromURL returns a new *redigo/redis.Pool configured for the supplied url
// The url can include a password in the standard form and if so is used to AUTH against
// the redis server
func NewRedisPoolFromURL(rawURL string, altPasses ...string) (*redis.Pool, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	var (
		mu       sync.Mutex
		password string
	)

	// strip and save password
	if pass, ok := u.User.Password(); ok {
		password = pass
		altPasses = append(altPasses, pass)
	}

	// DialURL will error if wrong password is set. We want to create a
	// successful connection with which we can try all of the possible
	// passwords.
	u.User = url.UserPassword("", "")
	rawURL = u.String()

	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redisDialURL(rawURL)
			if err != nil {
				return nil, err
			}

			// Capture the current best known valid pass.
			mu.Lock()
			localPass := password
			mu.Unlock()

			var authErr error

			if _, authErr = c.Do("AUTH", localPass); authErr != nil {
				for _, pass := range altPasses {
					if pass == localPass {
						continue
					}

					if _, authErr = c.Do("AUTH", pass); authErr == nil {
						// nominate this pass as valid.
						mu.Lock()
						password = pass
						mu.Unlock()
						break
					}
				}
			}

			if authErr != nil {
				c.Close()
				return nil, authErr
			}

			return c, nil
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
