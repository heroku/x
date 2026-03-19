package service

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
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


func TestRedirectHandler(t *testing.T) {
	server := httptest.NewServer(redirectHandler(nil))
	defer server.Close()
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	serverURL.Scheme = "https"
	serverURL.Path = "/"

	tests := []struct {
		name    string
		url     string
		wantURL *url.URL
	}{
		{
			name:    "url without path",
			url:     server.URL,
			wantURL: serverURL,
		},
		{
			name:    "url with path",
			url:     server.URL + "/some/path",
			wantURL: serverURL.ResolveReference(&url.URL{Path: "/some/path"}),
		},
		{
			name:    "url with path and query",
			url:     server.URL + "/some/path?a=b&b=c",
			wantURL: serverURL.ResolveReference(&url.URL{Path: "/some/path", RawQuery: "a=b&b=c"}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, err := client.Get(test.url)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusMovedPermanently {
				t.Fatalf("got %d but want %d", resp.StatusCode, http.StatusMovedPermanently)
			}

			loc, err := resp.Location()
			if err != nil {
				t.Fatal(err)
			}

			if loc.String() != test.wantURL.String() {
				t.Fatalf("got redirect URL: %s want %s", loc, test.wantURL)
			}
		})
	}
}
