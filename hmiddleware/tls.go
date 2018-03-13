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

// EnsureTLS ensures all incoming requests identify as having been proxied via
// https from the upstream reverse proxy. The way that this uses to check relies
// on the `X-Forwarded-Proto` header which is not defined by any formal standard. 
// For more information on this header, see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-Proto.
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
