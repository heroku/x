package redis

import (
	"context"

	"github.com/heroku/metaas/cache"
)

// Encoder ...
type Encoder func(v interface{}) ([]byte, error)

// Decoder ...
type Decoder func([]byte) (interface{}, error)

// interface compliance checks
var (
	_ cache.Cache = Cache{}
)

// Cache ...
type Cache struct {
	Prefix  string
	Storage Storage
	Encoder Encoder
	Decoder Decoder
}

// Put ...
func (c Cache) Put(ctx context.Context, key string, value interface{}) error {
	buf, err := c.Encoder(value)
	if err != nil {
		return err
	}
	return c.Storage.Put(ctx, c.Prefix, key, buf)
}

// Get ...
func (c Cache) Get(ctx context.Context, key string) (interface{}, bool) {
	v, err := c.Storage.Get(ctx, c.Prefix, key)
	if err != nil {
		return nil, false
	}

	buf, err := c.Decoder(v)
	return buf, err == nil
}

// Delete ...
func (c Cache) Delete(ctx context.Context, key string) (bool, error) {
	ok, err := c.Storage.Delete(ctx, c.Prefix, key)
	return err == nil && ok, err
}
