package scrub

import (
	"net/url"
	"testing"
)

func mustParseURL(t *testing.T, val string) *url.URL {
	u, err := url.Parse(val)
	if err != nil {
		t.Fatal(err)
	}

	return u
}

func TestURL(t *testing.T) {
	for param := range RestrictedParams {
		t.Run(param, func(tt *testing.T) {
			u := urlMustParse(tt, "https://thisisnotadoma.in/login")
			q := u.Query()

			q.Set(param, "hunter2")
			u.RawQuery = q.Encode()

			sc := URL(u)
			scq := sc.Query()

			if val := scq.Get(param); val != scrubbedValue {
				tt.Fatalf("%s: want: %q, got: %q", param, scrubbedValue, val)
			}
		})
	}
}

func TestURLUserInfo(t *testing.T) {
	u := mustParseURL(t, "https://AzureDiamond:hunter2@thisisnotadoma.in/login")
	sc := URL(u)

	user := sc.User.Username()
	if user != "AzureDiamond" {
		t.Fatalf("sc.User.Username(): want: \"AzureDiamond\", got: %q", user)
	}

	pass, ok := sc.User.Password()
	if !ok {
		t.Fatalf("expected sc.User.Password to have a value.")
	}

	if pass != scrubbedValue {
		t.Fatalf("sc.User.Password(): want: %q, got: %q", scrubbedValue, pass)
	}
}
