package redigo

import (
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
)

func setup(t *testing.T, dialErr error) (*redigomock.Conn, DialURLFunc, func()) {
	conn := redigomock.NewConn()
	redisDialURL := func(_ string, options ...redis.DialOption) (redis.Conn, error) {
		if dialErr != nil {
			return nil, dialErr
		}
		return conn, nil
	}

	return conn, redisDialURL, func() {
		redisDialURL = redis.DialURL
	}
}

func TestPool(t *testing.T) {
	mock, dialURL, tearDown := setup(t, nil)
	defer tearDown()

	mock.Command("AUTH", "badpass").ExpectError(errors.New("Bad password"))
	mock.Command("AUTH", "goodpass")

	sut, err := NewRedisPoolFromURL("redis://h:badpass@localhost:6379",
		WithPasswords("goodpass"),
		WithDialURLFunc(dialURL),
	)

	if err != nil {
		t.Fatalf("got %q, want nil", err)
	}

	// hold on to this conn forever, never close. This forces the next
	// Get() to redial to get a new connection.
	c := sut.Get()
	if c.Err() != nil {
		t.Fatalf("got %q, want nil", c.Err())
	}

	mock.Clear()

	// If we get a new connection, only "goodpass" should be called for Auth,
	// unless that fails again. See "the GoodPassNowFails test"
	mock.Command("AUTH", "goodpass")

	c = sut.Get()
	if c.Err() != nil {
		t.Fatalf("got %q, want nil", c.Err())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("got %q, want nil", err)
	}
}

func TestPoolGoodPassNowFails(t *testing.T) {
	mock, dialURL, tearDown := setup(t, nil)
	defer tearDown()

	mock.Command("AUTH", "badpass").ExpectError(errors.New("Bad password"))
	mock.Command("AUTH", "goodpass")

	sut, err := NewRedisPoolFromURL("redis://h:badpass@localhost:6379",
		WithPasswords("goodpass"),
		WithDialURLFunc(dialURL),
	)
	if err != nil {
		t.Fatalf("got %q, want nil", err)
	}

	// hold on to this conn forever, never close. This forces the next
	// Get() to redial to get a new connection.
	c := sut.Get()
	if c.Err() != nil {
		t.Fatalf("got %q, want nil", c.Err())
	}

	mock.Clear()

	// "goodpass" now fails, which is what we used to authenticate successfully before.
	// But, we should try to auth with "badpass" again, which will now succeed.
	mock.Command("AUTH", "goodpass").ExpectError(errors.New("Bad Password"))
	mock.Command("AUTH", "badpass")

	c = sut.Get()
	if c.Err() != nil {
		t.Fatalf("got %q, want nil", c.Err())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("got %q, want nil", err)
	}
}

func TestPoolWithNoGoodPasses(t *testing.T) {
	mock, dialURL, tearDown := setup(t, nil)
	defer tearDown()

	mock.Command("AUTH", "badpass").ExpectError(errors.New("Bad password"))
	mock.Command("AUTH", "alsobadpass").ExpectError(errors.New("Bad password"))

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
}

// Note: If this test fails, it can leave the redis with a password different than you
// expect. Probably best to run redis with a fresh docker container, like so:
//
//    docker run --rm -p 127.0.0.1:6379:6379 --name test-redis redis
//    docker exec -ti test-redis redis-cli config set requirepass password
//    export REDIS_URL=redis://h:password@127.0.0.1:6379:6379
func TestForReals(t *testing.T) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skip()
	}

	u, err := url.Parse(redisURL)
	if err != nil {
		t.Fatalf("got %q, want nil", err)
	}

	initPassword, _ := u.User.Password()

	passes := []string{
		"one",
		"two",
		"three",
		"four",
		"five",
	}

	sut, err := NewRedisPoolFromURL(redisURL, WithPasswords(passes...))
	initConn := sut.Get()

	defer func() {
		if _, err = initConn.Do("CONFIG", "SET", "requirepass", initPassword); err != nil {
			t.Fatalf("got %q, want nil", err)
		}
	}()

	for _, newPass := range passes {
		c := sut.Get()
		if _, err = c.Do("CONFIG", "SET", "requirepass", newPass); err != nil {
			t.Fatalf("got %q, want nil", err)
		}

		time.Sleep(10 * time.Millisecond)
	}

	c := sut.Get()
	if err != nil {
		t.Fatalf("got %q, want nil", err)
	}

	// What's the current pass?
	if result, err := redis.Strings(c.Do("CONFIG", "GET", "requirepass")); err != nil {
		if len(result) == 2 {
			if result[1] == initPassword {
				t.Errorf("got %s, want NOT %s", result[1], initPassword)
			}
		} else {
			t.Errorf("got %d, want 2", len(result))
		}
	}
}
