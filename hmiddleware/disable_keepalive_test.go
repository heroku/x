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

func TestDisableKeepalive(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "everything is okay :)", http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rw := httptest.NewRecorder()

	DisableKeepalive(h).ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Logf("body: %s", string(rw.Body.Bytes()))
		t.Fatalf("expected rw.Code to be %d, got: %d", http.StatusOK, rw.Code)
	}

	if val := rw.Header().Get("Connection"); val != "close" {
		t.Fatalf("expected \"Connection: close\", got: \"Connection: %s\"", val)
	}
}
