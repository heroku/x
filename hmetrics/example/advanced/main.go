package main

/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/heroku/x/hmetrics"
)

func main() {
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
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
