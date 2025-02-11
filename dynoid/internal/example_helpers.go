package internal

import (
	"context"
	"sync"

	"github.com/heroku/x/dynoid/dynoidtest"
)

var (
	cfg = sync.OnceValue(func() *dynoidtest.LocalConfiguration {
		cfg, err := dynoidtest.ConfigureLocal([]string{})
		if err != nil {
			panic(err)
		}

		return cfg
	})
)

func GenerateToken(audience string) (context.Context, string) {
	token, err := cfg().GenerateToken(audience)
	if err != nil {
		panic(err)
	}

	return cfg().Context(), token
}
