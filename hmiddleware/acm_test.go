package hmiddleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestACMEValidationMiddleware(t *testing.T) {
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	validationURL := &url.URL{Scheme: "https", Host: "acm.local", Path: "/challenge", RawQuery: "foo=bar"}

	server := httptest.NewServer(ACMEValidationMiddleware(validationURL)(app))
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

	wantQuery := url.Values{
		"foo":   []string{"bar"},
		"token": []string{"some-token"},
		"host":  []string{serverURL.Host},
	}

	tests := []struct {
		name string
		url  string

		wantStatus int
		wantURL    *url.URL
	}{
		{
			name:       "no challenge path",
			url:        server.URL,
			wantStatus: http.StatusOK,
		},

		{
			name:       "with challenge path",
			url:        server.URL + http01ChallengePath + "some-token",
			wantStatus: http.StatusMovedPermanently,
			wantURL:    validationURL.ResolveReference(&url.URL{RawQuery: wantQuery.Encode()}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, err := client.Get(test.url)
			if err != nil {
				t.Fatal(err)
			}

			if test.wantStatus != resp.StatusCode {
				t.Fatalf("want code: %d got %d", test.wantStatus, resp.StatusCode)
			}

			if test.wantURL != nil {
				gotURL, err := resp.Location()
				if err != nil {
					t.Fatal(err)
				}

				if want, got := test.wantURL.String(), gotURL.String(); want != got {
					t.Fatalf("want location redirect: %s, got %s", want, got)
				}
			}

			if validationURL.RawQuery != "foo=bar" {
				t.Fatalf("want raw query to still be `foo=bar`, got %q", validationURL.RawQuery)
			}
		})
	}
}
