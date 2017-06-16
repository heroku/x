package redis

import (
	"context"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/go-kit/kit/metrics"
)

// Storage ...
type Storage interface {
	Get(ctx context.Context, prefix, key string) ([]byte, error)
	Put(ctx context.Context, prefix, key string, buf []byte) error
	Delete(ctx context.Context, prefix, key string) (bool, error)
}

// Hash ...
type Hash struct {
	Pool *redis.Pool

	// metrics
	PutTimes, GetTimes, DeleteTimes metrics.Histogram
}

func measure(h metrics.Histogram, start time.Time) {
	if h == nil { // guard against it not being set
		return
	}
	h.Observe(time.Since(start).Seconds())
}

// Put ...
func (h Hash) Put(ctx context.Context, prefix, key string, buf []byte) error {
	conn := h.Pool.Get()
	defer conn.Close()

	defer measure(h.PutTimes, time.Now())
	_, err := conn.Do("HSET", prefix, key, buf)
	return err
}

// Get ...
func (h Hash) Get(ctx context.Context, prefix, key string) ([]byte, error) {
	conn := h.Pool.Get()
	defer conn.Close()

	defer measure(h.GetTimes, time.Now())
	return redis.Bytes(conn.Do("HGET", prefix, key))
}

// Delete ...
func (h Hash) Delete(ctx context.Context, prefix, key string) (bool, error) {
	conn := h.Pool.Get()
	defer conn.Close()

	defer measure(h.DeleteTimes, time.Now())
	return redis.Bool(conn.Do("HDEL", prefix, key))
}

// Volatile ...
type Volatile struct {
	TTL  time.Duration
	Pool *redis.Pool

	// metrics
	PutTimes, GetTimes, DeleteTimes metrics.Histogram
}

// Put ...
func (v Volatile) Put(ctx context.Context, prefix, key string, buf []byte) error {
	conn := v.Pool.Get()
	defer conn.Close()

	defer measure(v.PutTimes, time.Now())
	_, err := conn.Do("PSETEX", prefix+":"+key, int(v.TTL.Seconds()*1000), buf)
	return err
}

// Get ...
func (v Volatile) Get(ctx context.Context, prefix, key string) ([]byte, error) {
	conn := v.Pool.Get()
	defer conn.Close()

	defer measure(v.GetTimes, time.Now())
	return redis.Bytes(conn.Do("GET", prefix+":"+key))
}

// Delete ...
func (v Volatile) Delete(ctx context.Context, prefix, key string) (bool, error) {
	conn := v.Pool.Get()
	defer conn.Close()

	defer measure(v.DeleteTimes, time.Now())
	return redis.Bool(conn.Do("DEL", prefix, key))
}
