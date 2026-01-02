package dynoid_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/heroku/x/dynoid"
	"github.com/heroku/x/dynoid/dynoidtest"
)

func TestVerification(t *testing.T) {
	ctx, token := GenerateIDToken(t, "heroku")

	verifier := dynoid.NewWithCallback("heroku", dynoid.AllowHerokuHost(dynoidtest.DefaultHerokuHost))

	if _, err := verifier.Verify(ctx, token); err != nil {
		t.Error(err)
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	in := dynoid.Subject{
		AppID:   "7eeecc9f-b17f-4027-9aa1-ceb8427036c6",
		AppName: "testing",
		Dyno:    "web.1",
	}

	var out dynoid.Subject

	if err := out.UnmarshalText([]byte(in.String())); err != nil {
		t.Fatalf("failed to unmarshal (%v)", err)
	}

	if out.AppID != in.AppID {
		t.Fatalf("AppID missmatch (%q != %q)", out.AppID, in.AppID)
	}

	if out.AppName != in.AppName {
		t.Fatalf("AppName missmatch (%q != %q)", out.AppName, in.AppName)
	}

	if out.Dyno != in.Dyno {
		t.Fatalf("Dyno missmatch (%q != %q)", out.Dyno, in.Dyno)
	}
}

func TestLocalTokenPath(t *testing.T) {
	// default audience uses default path
	if got := dynoid.LocalTokenPath("heroku"); got != "/etc/heroku/dyno_id_token" {
		t.Fatalf("unexpected path for heroku: %q", got)
	}

	// non-default audience uses audience-specific path
	if got := dynoid.LocalTokenPath("other"); got != "/etc/heroku/dyno-id/other/token" {
		t.Fatalf("unexpected path for other: %q", got)
	}

	// env var override
	t.Setenv("OTHER_IDENTITY_TOKEN_FILE", "/custom/path")
	if got := dynoid.LocalTokenPath("other"); got != "/custom/path" {
		t.Fatalf("expected env var override, got: %q", got)
	}
}

func TestReading(t *testing.T) {
	oldFS := dynoid.DefaultFS
	defer func() {
		dynoid.DefaultFS = oldFS
	}()

	spaceID := uuid.NewString()
	appID := uuid.NewString()

	ctx, tk := GenerateIDToken(t, "heroku",
		dynoidtest.WithSpaceID(spaceID),
		dynoidtest.WithTokenOpts(dynoidtest.WithSubject(&dynoid.Subject{
			AppID:   appID,
			AppName: "testapp",
			Dyno:    "run.1",
		})),
	)
	dynoid.DefaultFS = dynoidtest.NewFS(map[string]string{
		"heroku": tk,
	})

	token, err := dynoid.ReadLocalToken(ctx, "heroku")
	if err != nil {
		t.Fatalf("failed to read token (%v)", err)
	}

	if token.SpaceID != spaceID {
		t.Fatalf("Unexpected SpaceID %q", token.SpaceID)
	}

	if token.Subject.AppID != appID {
		t.Fatalf("Unexpected AppID %q", token.Subject.AppID)
	}
}

func GenerateIDToken(t *testing.T, audience string, opts ...dynoidtest.IssuerOpt) (context.Context, string) {
	t.Helper()

	ctx, iss, err := dynoidtest.NewWithContext(context.Background(), opts...)
	if err != nil {
		t.Fatal(err)
	}

	token, err := iss.GenerateIDToken(audience)
	if err != nil {
		t.Fatal(err)
	}

	return ctx, token
}
