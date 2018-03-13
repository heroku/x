/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import (
	"net/http"
)

// DisableKeepalive instructs the Go HTTP stack to close the incoming HTTP
// connection once all requests processed by this middleware are complete.
func DisableKeepalive(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Connection", "close")
		next.ServeHTTP(w, r)
	})
}
