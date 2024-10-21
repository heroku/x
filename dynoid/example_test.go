package dynoid_test

import (
	"fmt"

	"github.com/heroku/x/dynoid"
	"github.com/heroku/x/dynoid/internal"
)

const AUDIENCE = "testing"

func ExampleVerifier() {
	// Normally a token would be passed in, but for testing we'll generate one
	ctx, token := internal.GenerateToken(AUDIENCE)

	verifier := dynoid.New(AUDIENCE)
	verifier.IssuerCallback = dynoid.AllowHerokuHost("heroku.local") // heroku.com for production

	t, err := verifier.Verify(ctx, token)
	if err != nil {
		fmt.Printf("failed to verify token (%v)", err)
		return
	}

	fmt.Println(t.Subject.AppID)
	fmt.Println(t.Subject.AppName)
	fmt.Println(t.Subject.Dyno)
	// Output:
	// 00000000-0000-0000-0000-000000000001
	// sushi
	// web.1
}
