/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import (
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

func ExampleCORS() {
	r := chi.NewRouter()
	r.Use(CORS)

	// OR

	var h http.Handler

	http.ListenAndServe(":"+os.Getenv("PORT"), CORS(h))
}

func ExampleDisableKeepalive() {
	r := chi.NewRouter()
	r.Use(DisableKeepalive)

	// OR

	var h http.Handler

	http.ListenAndServe(":"+os.Getenv("PORT"), DisableKeepalive(h))
}

func ExampleEnsureTLS() {
	r := chi.NewRouter()
	r.Use(EnsureTLS)

	// OR

	var h http.Handler

	http.ListenAndServe(":"+os.Getenv("PORT"), EnsureTLS(h))
}
