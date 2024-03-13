package dynoid

import (
	"context"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/heroku/x/dynoid/dynoidtest"
)

func TestVerification(t *testing.T) {
	iss, err := dynoidtest.New()
	if err != nil {
		t.Fatal(err)
	}

	token, err := iss.GenerateIDToken("heroku")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = oidc.ClientContext(ctx, iss.HTTPClient())

	verifier := NewWithCallback("heroku", AllowHerokuHost("heroku.local"))

	if _, err = verifier.Verify(ctx, token); err != nil {
		t.Error(err)
	}
}
