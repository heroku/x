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
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestFromRequest(t *testing.T) {
	for _, h := range headersToSearch {
		t.Run(h, func(t *testing.T) {
			cases := []struct {
				name   string
				doer   func() *http.Request
				wantOK bool
			}{
				{
					name: "with request id header set",
					doer: func() *http.Request {
						req := httptest.NewRequest("GET", "/", nil)
						req.Header.Set(h, uuid.New().String())
						return req
					},
					wantOK: true,
				},
				{
					name:   "without request id header set",
					doer:   func() *http.Request { return httptest.NewRequest("GET", "/", nil) },
					wantOK: false,
				},
			}

			for _, cs := range cases {
				t.Run(cs.name, func(t *testing.T) {
					_, ok := FromRequest(cs.doer())

					if !ok && cs.wantOK {
						t.Fatalf("expected to fetch request ID, but couldn't")
					}
				})
			}
		})
	}
}

func TestFromRequest_AppendsIncomingRequestID(t *testing.T) {
	originalRequestID := uuid.New().String()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", originalRequestID)
	requestID, ok := FromRequest(req)
	requestIDs := strings.Split(requestID, ",")
	reqIDInHeader := req.Header.Get("X-Request-Id")

	if !ok {
		t.Fatalf("no RequestID found in Headers")
	}

	if len(requestIDs) != 2 {
		t.Fatalf("got %v Request IDs, want 2", len(requestIDs))
	}

	if requestIDs[1] != originalRequestID {
		t.Fatalf("second Request ID was %v, want %v", requestIDs[1], originalRequestID)
	}

	if reqIDInHeader != requestID {
		t.Fatalf("request ID in header was %v, want %v", req.Header.Get("X-Request-Id"), requestID)
	}
}

func TestRequestIDFromContext(t *testing.T) {
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
