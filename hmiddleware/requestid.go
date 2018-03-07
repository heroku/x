/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import (
	"net/http"

	"github.com/heroku/metaas/context/requestid"
	"github.com/heroku/x/hcontext"
)

// RequestID extracts, or creates a request ID and adds it to the context
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID, _ := hcontext.FromRequest(r)
		w.Header().Add("X-Request-Id", reqID) // give request ID to user so things can be debugged easier
		next.ServeHTTP(w, r.WithContext(requestid.WithRequestID(r.Context(), reqID)))
	})
}
