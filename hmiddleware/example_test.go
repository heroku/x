/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hmiddleware

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

func ExampleCORS() {
	r := chi.NewRouter()
	r.Use(CORS)

	// OR

	var h http.Handler

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), CORS(h)); err != nil {
		log.Fatal(err)
	}
}

func ExampleDisableKeepalive() {
	r := chi.NewRouter()
	r.Use(DisableKeepalive)

	// OR

	var h http.Handler

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), DisableKeepalive(h)); err != nil {
		log.Fatal(err)
	}
}

func ExampleEnsureTLS() {
	r := chi.NewRouter()
	r.Use(EnsureTLS)

	// OR

	var h http.Handler

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), EnsureTLS(h)); err != nil {
		log.Fatal(err)
	}
}
