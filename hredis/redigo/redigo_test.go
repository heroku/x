package redigo

import (
	"errors"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
)

func setup(t *testing.T, dialErr error) (*redigomock.Conn, func()) {
	conn := redigomock.NewConn()
	redisDialURL = func(_ string, options ...redis.DialOption) (redis.Conn, error) {
		if dialErr != nil {
			return nil, dialErr
		}
		return conn, nil
	}

	return conn, func() {
		redisDialURL = redis.DialURL
	}
}

func TestPool(t *testing.T) {
	mock, tearDown := setup(t, nil)
	defer tearDown()

	mock.Command("AUTH", "badpass").ExpectError(errors.New("Bad password"))
	mock.Command("AUTH", "goodpass").Expect("OK")

	sut, err := NewRedisPoolFromURL("redis://h:badpass@localhost:6379", "goodpass")
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

	// If we get a new connection, only "goodpass" should be called for Auth
	mock.Command("AUTH", "goodpass")

	c = sut.Get()
	if c.Err() != nil {
		t.Fatalf("got %q, want nil", c.Err())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("got %q, want nil", err)
	}
}

func TestPoolWithNoGoodPasses(t *testing.T) {
	mock, tearDown := setup(t, nil)
	defer tearDown()

	mock.Command("AUTH", "badpass").ExpectError(errors.New("Bad password"))
	mock.Command("AUTH", "alsobadpass").ExpectError(errors.New("Bad password"))

	sut, err := NewRedisPoolFromURL("redis://h:badpass@localhost:6379", "alsobadpass")
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
