/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hcontext

import (
	"log"
	"net/http"
)

func ExampleFromRequest() {
	var r *http.Request

	reqID, ok := FromRequest(r)
	if !ok {
		log.Printf("when handling request from %s, no request ID", r.RemoteAddr)
		return
	}

	log.Printf("The request ID is: %s", reqID)
}

func ExampleWithRequestID() {
	var r *http.Request

	reqID, ok := FromRequest(r)
	if !ok {
		log.Printf("when handling request from %s, no request ID", r.RemoteAddr)
		return
	}

	ctx := WithRequestID(r.Context(), reqID)
	r = r.WithContext(ctx)
}
