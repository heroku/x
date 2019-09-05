package https

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// redirectHandler returns a handler that redirects all requests to HTTPS.
func TestRedirectHandler(t *testing.T) {
	server := httptest.NewServer(RedirectHandler(nil))
	defer server.Close()
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
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
		name string
		url  string

		wantURL *url.URL
	}{
		{
			name: "url without path",
			url:  server.URL,

			wantURL: serverURL,
		},
		{
			name: "url with path",
			url:  server.URL + "/some/path",

			wantURL: serverURL.ResolveReference(&url.URL{Path: "/some/path"}),
		},
		{
			name: "url with path and query",
			url:  server.URL + "/some/path?a=b&b=c",

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

			url, err := resp.Location()
			if err != nil {
				t.Fatal(err)
			}

			if url.String() != test.wantURL.String() {
				t.Fatalf("got redirect URL: %s want %s", url, test.wantURL)
			}
		})
	}
}
