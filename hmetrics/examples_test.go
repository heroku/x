/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */
package hmetrics_test

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/heroku/x/hmetrics"
)

func ExampleReport_basic() {
	// Don't care about canceling or errors
	go hmetrics.Report(context.Background(), hmetrics.DefaultEndpoint, nil)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, nil)
}

func ExampleReport_logging() {
	go func() {
		if err := hmetrics.Report(context.Background(), hmetrics.DefaultEndpoint, func(err error) error {
			log.Println("Error reporting metrics to heroku:", err)
			return nil
		}); err != nil {
			log.Fatal("Error starting hmetrics reporting:", err)
		}
	}()
}
func ExampleReport_advanced() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		type fataler interface {
			Fatal() bool
		}
		for { // try again and again on non fatal errors
			if err := hmetrics.Report(ctx, hmetrics.DefaultEndpoint, func(err error) error {
				log.Println("Error reporting metrics to heroku:", err)
				return nil
			}); err != nil {
				if f, ok := err.(fataler); ok && f.Fatal() {
					log.Fatal(err)
				}
				log.Println(err)
			}
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, nil)
}
