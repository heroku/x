package hredis

import "testing"

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
			_, err := RedissURL(cs.url)

			if err == nil && cs.shouldErr {
				t.Fatal("wanted non-nil error but got nil error")
			}

			if err != nil && !cs.shouldErr {
				t.Fatalf("wanted nil error but got: %v", err)
			}
		})
	}
}
