/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import "net/http"

const (
	allowMethods = "POST, GET, OPTIONS, PUT, DELETE, PATCH"
	allowHeaders = "Location, X-Request-ID"
)

// CORS adds Cross-Origin Resource Sharing headers to all outgoing requests.
// This is known as something that is kind of hard to get right. See docs at
// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS for more information.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", allowMethods)
		w.Header().Set("Access-Control-Expose-Headers", allowHeaders)
		w.Header()["Access-Control-Allow-Headers"] = r.Header["Access-Control-Request-Headers"]
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
