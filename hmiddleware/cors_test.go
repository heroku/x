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

func TestCORSHeaders(t *testing.T) {
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some-url", nil)

	req.Header.Add("Origin", "some-origin.com")
	req.Header.Add("Access-Control-Request-Headers", "Content-Type")
	req.Header.Add("Access-Control-Request-Headers", "Accept")

	CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(resp, req)

	aes := func(what, a, b string) {
		if a != b {
			t.Fatalf("expected %s to be %s, but got: %s", what, a, b)
		}
	}
	aess := func(what string, a, b []string) {
		if len(a) != len(b) {
			t.Fatalf("expected %s to have the same length, got: %v, %v", what, a, b)
		}

		for i := range a {
			aa := a[i]
			bb := b[i]

			if aa != bb {
				t.Fatalf("expected %s to have the same value at index %d, it didn't: %v vs %v", what, i, aa, bb)
			}
		}
	}

	aes("Access-Control-Allow-Origin header in response", "some-origin.com", resp.Header().Get("Access-Control-Allow-Origin"))
	aes("Access-Control-Allow-Methods header in response", allowMethods, resp.Header().Get("Access-Control-Allow-Methods"))
	aess("Access-Control-Allow-Headers header in response", []string{"Content-Type", "Accept"}, resp.Header()["Access-Control-Allow-Headers"])
}

func TestCORSShortCircuitOnOptions(t *testing.T) {
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/some-url", nil)

	CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})).ServeHTTP(resp, req)

	// Make sure we actually don't hit the 500 inside the block, and get the 200 short circuit.
	if resp.Code != http.StatusOK {
		t.Fatalf("wanted response code of %d, got: %d", http.StatusOK, resp.Code)
	}
}
