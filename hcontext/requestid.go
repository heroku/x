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

var key idkey

var headersToSearch = []string{
	"Request-ID", "X-Request-ID",
}

// FromRequest ...
func FromRequest(r *http.Request) (id string, ok bool) {
	for _, try := range headersToSearch {
		if id = r.Header.Get(try); id != "" {
			return id, true
		}
	}

	return uuid.NewRandom().String(), false
}

// WithRequestID ...
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key, id)
}

// FromContext ...
func FromContext(ctx context.Context) (id string, ok bool) {
	id, ok = ctx.Value(key).(string)
	return
}
