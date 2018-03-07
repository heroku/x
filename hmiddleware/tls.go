/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import (
	"fmt"
	"net/http"
)

const maxAge = 31536000 // 1 year in seconds.

// EnsureTLS ...
func EnsureTLS(next http.Handler) http.Handler {
	hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains", maxAge)

	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", hstsValue)
		if r.URL.Scheme != "https" && r.Header.Get("X-Forwarded-Proto") != "https" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
