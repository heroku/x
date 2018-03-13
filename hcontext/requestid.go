/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hcontext

import (
	"context"
	"net/http"

	"github.com/pborman/uuid"
)

type idkey int

var ridKey idkey

var headersToSearch = []string{
	"Request-Id", "X-Request-Id",
	"Request-ID", "X-Request-ID",
}

// FromRequest fetches the given request's request ID if it has one,
// and returns a new random request ID if it does not.
func FromRequest(r *http.Request) (id string, ok bool) {
	for _, try := range headersToSearch {
		if id = r.Header.Get(try); id != "" {
			return id, true
		}
	}

	newRID := uuid.New()
	r.Header.Set("X-Request-Id", newRID)
	return newRID, false
}

// WithRequestID adds the given request ID to a context for processing later
// down the chain.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ridKey, id)
}

// RequestIDFromContext fetches a request ID from the given context if it exists.
func RequestIDFromContext(ctx context.Context) (id string, ok bool) {
	id, ok = ctx.Value(ridKey).(string)
	return
}
