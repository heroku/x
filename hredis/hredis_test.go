/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hredis

import (
	"net/url"
	"testing"
)

func TestRedissURL(t *testing.T) {
	cases := []struct {
		url       string
		shouldErr bool
	}{
		{url: "redis://ad:hunter2@127.0.0.1:80", shouldErr: false},
		{url: "rediss://ad:hunter2@127.0.0.1:80", shouldErr: false},
		{url: "http://google.com", shouldErr: true},
		{url: "redis://ad:hunter2@127.0.0.1:port", shouldErr: true},
	}

	for _, cs := range cases {
		t.Run(cs.url, func(t *testing.T) {
			u, err := RedissURL(cs.url)

			if err == nil && cs.shouldErr {
				t.Fatal("wanted non-nil error but got nil error")
			}

			if err != nil && !cs.shouldErr {
				t.Fatalf("wanted nil error but got: %v", err)
			}

			_, err = url.Parse(u)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
