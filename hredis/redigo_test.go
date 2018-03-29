/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package hredis

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/tidwall/redcon"
)

// dontWait for time to pass, we're a TARDIS or something.
func dontWait(t time.Time) error { return nil }

func makeFakeRedis(t *testing.T) (context.CancelFunc, *url.URL) {
	ctx, cancel := context.WithCancel(context.Background())

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	s := redcon.NewServer(addr, func(c redcon.Conn, cmd redcon.Command) {
		t.Logf("%s issued a command: %s", c.RemoteAddr(), string(cmd.Args[0]))
		c.WriteString("OK")
	}, func(c redcon.Conn) bool {
		t.Logf("connection from %s", c.RemoteAddr())
		return true
	}, func(c redcon.Conn, err error) {
		t.Logf("connection lost from %s: %v", c.RemoteAddr(), err)
	})
	t.Logf("fake redis running at redis://%s", addr)

	go s.ListenAndServe()
	go func(s *redcon.Server) {
		<-ctx.Done()
		s.Close()
	}(s)

	return cancel, &url.URL{
		Scheme: "redis",
		Host:   addr,
	}
}

func TestWaitForAvailability(t *testing.T) {
	cancel, rurl := makeFakeRedis(t)
	defer cancel()

	ok, err := WaitForAvailability(rurl.String(), time.Second, dontWait)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected for redis server to be available")
	}
}

func TestNewRedisPoolFromURL(t *testing.T) {
	cancel, rurl := makeFakeRedis(t)
	defer cancel()

	p, err := NewRedisPoolFromURL(rurl.String())
	if err != nil {
		t.Fatal(err)
	}

	conn := p.Get()
	defer conn.Close()

	val, err := conn.Do("HELLO", "WORLD")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("value from fake redis: %v", val)

	p.Close()
}
