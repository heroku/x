/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hcontext

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pborman/uuid"
)

func TestFromRequest(t *testing.T) {
	for _, h := range headersToSearch {
		t.Run(h, func(t *testing.T) {
			cases := []struct {
				name       string
				doer       func() *http.Request
				shouldWork bool
			}{
				{
					name: "everything works as normal",
					doer: func() *http.Request {
						req := httptest.NewRequest("GET", "/", nil)
						req.Header.Set(h, uuid.New())
						return req
					},
				},
				{
					name:       "everything doesn't work",
					doer:       func() *http.Request { return httptest.NewRequest("GET", "/", nil) },
					shouldWork: false,
				},
			}

			for _, cs := range cases {
				t.Run(cs.name, func(t *testing.T) {
					_, ok := FromRequest(cs.doer())
					if !ok && cs.shouldWork {
						t.Fatalf("expected to fetch request ID, but couldn't")
					}
				})
			}
		})
	}
}

func TestRequestIDStorage(t *testing.T) {
	const reqID = `hunter2`

	ctx := context.Background()
	ctx = WithRequestID(ctx, reqID)
	rid2, ok := RequestIDFromContext(ctx)
	if !ok {
		t.Fatalf("expected to get request ID from context but didn't")
	}

	if reqID != rid2 {
		t.Fatalf("expected to get %q from context, got: %q", reqID, rid2)
	}
}
