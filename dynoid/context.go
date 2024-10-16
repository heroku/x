package dynoid

import (
	"context"
	"errors"
)

var (
	ErrTokenNotSet = errors.New("token not set") // returned when neither a token nor an error is set
)

type dynoidCtxKey uint

const (
	tokenCtxKey dynoidCtxKey = iota
)

type maybeToken struct {
	*Token
	error
}

// ContextWithToken adds the given Token to the Context to be retrieved later
// by calling FromContext
func ContextWithToken(ctx context.Context, t *Token) context.Context {
	return context.WithValue(ctx, tokenCtxKey, &maybeToken{Token: t})
}

// ContextWithError adds the given error to the Context to be retrieved later
// by calling FromContext
func ContextWithError(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, tokenCtxKey, &maybeToken{error: err})
}

// FromContext returns the Token or error associated with the given Context
func FromContext(ctx context.Context) (*Token, error) {
	if v, ok := ctx.Value(tokenCtxKey).(*maybeToken); ok {
		return v.Token, v.error
	}

	return nil, ErrTokenNotSet
}
