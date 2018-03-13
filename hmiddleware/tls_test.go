/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnsureTLS(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "everything is okay :)", http.StatusOK)
	})

	cases := []struct {
		name     string
		h        http.Header
		wantCode int
	}{
		{
			name: "allow request with header",
			h: http.Header{
				"X-Forwarded-Proto": []string{"https"},
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "forbid request without header",
			h:        http.Header{},
			wantCode: http.StatusForbidden,
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			rw := httptest.NewRecorder()

			req.Header = cs.h

			EnsureTLS(h).ServeHTTP(rw, req)

			if rw.Code != cs.wantCode {
				t.Fatalf("response code was %d, wanted: %d", rw.Code, cs.wantCode)
			}
		})
	}
}
