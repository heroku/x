package service

import (
	"io"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/heroku/x/testing/testlog/v2"
)

func TestStandardHTTPServer(t *testing.T) {
	l, _ := testlog.New()
	//nolint: gosec
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	data, _ := io.ReadAll(res.Body)
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
	HTTP(l, nil, WithHTTPServerHook(config))

	if len(configuredServers) != 2 {
		t.Fatalf("expected 2 servers to be configured, got %v", configuredServers)
	}
}

