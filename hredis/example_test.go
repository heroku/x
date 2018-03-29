/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hredis

import (
	"log"
	"os"
	"time"
)

func ExampleRedissURL() {
	u, err := RedissURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// do something with the proper Redis URL `u`
	_ = u
}

func ExampleWaitForAvailability() {
	wait := func(t time.Time) error {
		log.Printf("couldn't connect to redis, trying again...")
		time.Sleep(time.Second)
		return nil
	}

	ok, err := WaitForAvailability(os.Getenv("REDIS_URL"), time.Minute, wait)
	if err != nil {
		log.Fatal(err)
	}

	if !ok {
		log.Fatal("can't connect to redis due to a timeout, check connection?")
	}
}

func ExampleNewRedisPoolFromURL() {
	p, err := NewRedisPoolFromURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// do something with pool 'p'
	p.Close()
}
