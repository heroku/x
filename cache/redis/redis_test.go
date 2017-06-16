package redis

import (
	"context"
	"testing"
	"time"

	"github.com/heroku/metaas/config"
	"github.com/heroku/metaas/internal/redis"
	"github.com/stretchr/testify/assert"
)

func TestGetPutDelHash(t *testing.T) {
	if config.Redis.URL == "" {
		t.Skip("Skipping because there's no REDIS URL")
	}

	pool, err := redis.NewPool(config.Redis.URL)
	if err != nil {
		t.Fatal(err)
	}

	c := Cache{Storage: Hash{Pool: pool}, Prefix: "monitor", Encoder: StringEncoder, Decoder: StringDecoder}

	_, ok := c.Get(context.Background(), "hello")
	assert.False(t, ok)

	err = c.Put(context.Background(), "hello", "world")
	assert.Nil(t, err)

	v, ok := c.Get(context.Background(), "hello")
	assert.True(t, ok)
	assert.EqualValues(t, "world", v)

	ok, err = c.Delete(context.Background(), "hello")
	assert.True(t, ok)
	assert.Nil(t, err)
}

func TestGetPutDelTTL(t *testing.T) {
	if config.Redis.URL == "" {
		t.Skip("Skipping because there's no REDIS URL")
	}

	pool, err := redis.NewPool(config.Redis.URL)
	if err != nil {
		t.Fatal(err)
	}

	c := Cache{Storage: Volatile{Pool: pool, TTL: 5 * time.Millisecond}, Prefix: "monitor", Encoder: StringEncoder, Decoder: StringDecoder}

	_, ok := c.Get(context.Background(), "hello")
	assert.False(t, ok)

	err = c.Put(context.Background(), "hello", "world")
	assert.Nil(t, err)

	v, ok := c.Get(context.Background(), "hello")
	assert.True(t, ok)
	assert.EqualValues(t, "world", v)

	ok, err = c.Delete(context.Background(), "hello")
	assert.True(t, ok)
	assert.Nil(t, err)

	err = c.Put(context.Background(), "hello", "world")
	assert.Nil(t, err)

	// It should expire after this
	time.Sleep(time.Millisecond * 10)
	_, ok = c.Get(context.Background(), "hello")
	assert.False(t, ok)
}
