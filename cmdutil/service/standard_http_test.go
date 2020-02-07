package service

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
	"github.com/heroku/x/testing/testlog"
)

func TestStandardHTTPServer(t *testing.T) {
	l, _ := testlog.New()
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.WriteString(w, "OK"); err != nil {
				t.Error(err)
			}
		}),
		Addr: "127.0.0.1:0",
	}

	listenHook = make(chan net.Listener)
	defer func() { listenHook = nil }()

	s := standardServer(l, srv)

	done := make(chan struct{})
	go func() {
		if err := s.Run(); err != nil {
			t.Log(err)
		}
		close(done)
	}()

	addr := (<-listenHook).Addr().String()

	res, err := http.Get("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	data, _ := ioutil.ReadAll(res.Body)
	if string(data) != "OK" {
		t.Fatalf("want OK got %v", string(data))
	}

	s.Stop(nil)

	<-done
}

func TestBypassHTTPServer(t *testing.T) {
	l, _ := testlog.New()
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.WriteString(w, "OK"); err != nil {
				t.Error(err)
			}
		}),
		Addr: "127.0.0.1:0",
	}

	listenHook = make(chan net.Listener)
	defer func() { listenHook = nil }()

	s := bypassServer(l, srv)

	done := make(chan struct{})
	go func() {
		if err := s.Run(); err != nil {
			t.Log(err)
		}
		close(done)
	}()

	addr := (<-listenHook).Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = io.WriteString(conn, "PROXY TCP4 127.0.0.1 127.0.0.1 44444 55555\n")
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("GET", "http://"+addr, nil)
	if err := req.Write(conn); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(conn)
	res, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	data, _ := ioutil.ReadAll(res.Body)
	if string(data) != "OK" {
		t.Fatalf("want OK got %v", string(data))
	}

	s.Stop(nil)

	<-done
}

func TestHTTPServerConfiguration(t *testing.T) {
	os.Setenv("PORT", "1234")
	os.Setenv("ADDITIONAL_PORT", "4567")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ADDITIONAL_PORT")
	}()

	var configuredServers []string
	config := func(s *http.Server) {
		configuredServers = append(configuredServers, s.Addr)
	}

	l, _ := testlog.New()
	p := testmetrics.NewProvider(t)
	HTTP(l, p, nil, WithHTTPServerHook(config))

	if len(configuredServers) != 2 {
		t.Fatalf("expected 2 servers to be configured, got %v", configuredServers)
	}
}
