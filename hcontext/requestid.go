/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hcontext

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type idkey int

var ridKey idkey

var headersToSearch = []string{
	"Request-Id", "X-Request-Id",
	"Request-ID", "X-Request-ID",
}

// FromRequest fetches the given request's request ID from the Headers.
// If one is found, it appends a new request ID and sets the comma separated value as the header.
// If one is not found, it sets a new request ID as the header.
func FromRequest(r *http.Request) (id string, ok bool) {
	newRID := uuid.New().String()

	for _, try := range headersToSearch {
		if id = r.Header.Get(try); id != "" {
			newRID = fmt.Sprintf("%s,%s", newRID, id)
			r.Header.Set("X-Request-Id", newRID)
			return newRID, true
		}
	}

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
