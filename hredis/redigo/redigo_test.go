package redigo

import (
	"errors"
	"os"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
)

func setup(t *testing.T) (*redigomock.Conn, DialURLFunc, func()) {
	t.Helper()
	conn := redigomock.NewConn()
	redisDialURL := func(_ string, _ ...redis.DialOption) (redis.Conn, error) { //nolint:unparam
		var err error
		return conn, err
	}

	return conn, redisDialURL, func() {
		redisDialURL = redis.DialURL
	}
}

var (
	errInvalidPassword = errors.New("ERR invalid password")
)

func TestPoolBadThenGood(t *testing.T) {
	mock, dialURL, tearDown := setup(t)
	defer tearDown()

	mock.Command("AUTH", "badpass").ExpectError(errInvalidPassword)
	mock.Command("AUTH", "goodpass").Expect("OK")

	sut, err := NewRedisPoolFromURL("redis://h:badpass@localhost:6379",
		WithPasswords("goodpass"),
		WithDialURLFunc(dialURL),
	)
	if err != nil {
		t.Fatalf("got %q, want nil", err)
	}

	c := sut.Get()
	if c.Err() != nil {
		t.Fatalf("got %q, want nil", c.Err())
	}
	c.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("got %q, want nil", err)
	}
}

func TestPoolGoodThenBad(t *testing.T) {
	mock, dialURL, tearDown := setup(t)
	defer tearDown()

	mock.Command("AUTH", "goodpass").Expect("OK")

	sut, err := NewRedisPoolFromURL("redis://h:goodpass@localhost:6379",
		WithPasswords("badpass"), // won't be used, since goodpass works
		WithDialURLFunc(dialURL),
	)
	if err != nil {
		t.Fatalf("got %q, want nil", err)
	}

	c := sut.Get()
	if c.Err() != nil {
		t.Fatalf("got %q, want nil", c.Err())
	}
	c.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("got %q, want nil", err)
	}
}

func TestPoolWithNoGoodPasses(t *testing.T) {
	mock, dialURL, tearDown := setup(t)
	defer tearDown()

	mock.Command("AUTH", "badpass").ExpectError(errInvalidPassword)
	mock.Command("AUTH", "alsobadpass").ExpectError(errInvalidPassword)

	sut, err := NewRedisPoolFromURL("redis://h:badpass@localhost:6379",
		WithPasswords("alsobadpass"),
		WithDialURLFunc(dialURL),
	)
	if err != nil {
		t.Fatalf("got %q, want nil", err)
	}

	// hold on to this conn forever, never close. This forces the next
	// Get() to redial to get a new connection.
	c := sut.Get()
	if c.Err() == nil {
		t.Fatal("got nil, want err")
	}
	c.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("got %q, want nil", err)
	}

}

// Note: If this test fails, it can leave the redis with a password different than you
// expect. Probably best to run redis with a fresh docker container, like so:
//
//    docker run --rm -p 127.0.0.1:6379:6379 --name test-redis redis
//    REDIS_URL=redis://127.0.0.1:6379 go test -count=1 -v ./hredis/...
func TestIntegrationTest(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip()
	}

	mp, err := NewRedisPoolFromURL(redisURL)
	if err != nil {
		t.Fatal("expected nil, got ", err)
	}
	defer mp.Close()

	mconn := mp.Get()
	defer mconn.Close()

	_, err = mconn.Do("PING")
	if err != nil {
		t.Fatal("expected nil, got ", err)
	}

	passwords := []string{
		"one",
		"two",
		"three",
		"four",
		"five",
	}

	var cpwd string
	for _, pwd := range passwords {
		t.Run(pwd, func(t *testing.T) {
			if cpwd != "" {
				if _, err := mconn.Do("AUTH", cpwd); err != nil {
					t.Fatal("expected nil, got ", err)
				}
			}
			if _, err := mconn.Do("CONFIG", "SET", "requirepass", pwd); err != nil {
				t.Fatal("expected nil, got ", err)
			}
			cpwd = pwd

			p, err := NewRedisPoolFromURL(redisURL, WithPasswords(passwords...))
			if err != nil {
				t.Fatal("expected nil, got ", err)
			}

			c := p.Get()
			if _, err := c.Do("PING"); err != nil {
				t.Fatal("expected nil, got ", err)
			}
			c.Close()
		})
	}

	defer func() {
		// reset the password to none
		if _, err := mconn.Do("CONFIG", "SET", "requirepass", ""); err != nil {
			t.Fatal("expected nil, got ", err)
		}
	}()
}
