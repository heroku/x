/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */
package main

import (
	"context"
	"net/http"
	"os"

	"github.com/heroku/x/hmetrics"
)

func main() {
	// Don't care about canceling or errors
	go hmetrics.Report(context.Background(), hmetrics.DefaultEndpoint, nil)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, nil)
}
