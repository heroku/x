package dynoid

import (
	"context"
	"testing"

	"github.com/heroku/x/dynoid/dynoidtest"
)

func TestVerification(t *testing.T) {
	ctx, iss, err := dynoidtest.NewWithContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	token, err := iss.GenerateIDToken("heroku")
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewWithCallback("heroku", AllowHerokuHost(dynoidtest.IssuerHost))

	if _, err = verifier.Verify(ctx, token); err != nil {
		t.Error(err)
	}
}
