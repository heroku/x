package dynoidtest_test

import (
	"context"
	"fmt"

	"github.com/heroku/x/dynoid"
	"github.com/heroku/x/dynoid/dynoidtest"
)

const Audience = "testing"

func ExampleIssuer() {
	ctx, iss, err := dynoidtest.NewWithContext(context.Background())
	if err != nil {
		panic(err)
	}

	if err := dynoidtest.GenerateDefaultFS(iss, Audience); err != nil {
		panic(err)
	}

	token, err := dynoid.ReadLocalToken(ctx, Audience)
	if err != nil {
		panic(err)
	}

	fmt.Println(token.Subject.AppID)
	fmt.Println(token.Subject.AppName)
	fmt.Println(token.Subject.Dyno)
	// Output:
	// 00000000-0000-0000-0000-000000000001
	// sushi
	// web.1
}
