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

	verifier := New("heroku")
	verifier.IssuerCallback = AllowHerokuHost("heroku.local")

	_, err = verifier.Verify(ctx, token)
	if err != nil {
		t.Error(err)
	}
}
