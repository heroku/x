package dynoid

import (
	"testing"

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

	verifier := NewWithCallback("heroku", AllowHerokuHost(dynoidtest.IssuerHost))

	ctx := iss.Context()
	if _, err = verifier.Verify(ctx, token); err != nil {
		t.Error(err)
	}
}
