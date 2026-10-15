package dynoid_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
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

func TestVerifyIDTokenServiceSubject(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _, err := dynoidtest.NewWithContext(context.Background(), dynoidtest.WithKey(key), dynoidtest.WithSpaceID("space-1"))
	if err != nil {
		t.Fatal(err)
	}

	sign := func(exp time.Duration) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
			Issuer:    "https://oidc." + dynoidtest.DefaultHerokuHost + "/spaces/space-1",
			Subject:   "system:serviceaccount:observability:collector",
			Audience:  jwt.ClaimStrings{"svc"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
		})
		tok.Header["kid"] = "primary"
		raw, err := tok.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}

	verifier := dynoid.NewWithCallback("svc", dynoid.AllowHerokuSpace(dynoidtest.DefaultHerokuHost, "space-1"))

	t.Run("accepts a non-dyno subject", func(t *testing.T) {
		raw := sign(time.Minute)
		if _, err := verifier.Verify(ctx, raw); err == nil {
			t.Error("Verify must still reject a subject that is not a dyno")
		}
		tok, err := verifier.VerifyIDToken(ctx, raw)
		if err != nil {
			t.Fatal(err)
		}
		if tok.Subject != "system:serviceaccount:observability:collector" {
			t.Errorf("subject = %q", tok.Subject)
		}
	})

	t.Run("rejects expired tokens", func(t *testing.T) {
		if _, err := verifier.VerifyIDToken(ctx, sign(-time.Minute)); err == nil {
			t.Error("expected expired token to be rejected")
		}
	})

	t.Run("rejects a wrong audience", func(t *testing.T) {
		other := dynoid.NewWithCallback("other", dynoid.AllowHerokuSpace(dynoidtest.DefaultHerokuHost, "space-1"))
		if _, err := other.VerifyIDToken(ctx, sign(time.Minute)); err == nil {
			t.Error("expected wrong audience to be rejected")
		}
	})

	t.Run("rejects an untrusted issuer", func(t *testing.T) {
		other := dynoid.NewWithCallback("svc", dynoid.AllowHerokuSpace(dynoidtest.DefaultHerokuHost, "space-2"))
		var untrusted *dynoid.UntrustedIssuerError
		if _, err := other.VerifyIDToken(ctx, sign(time.Minute)); !errors.As(err, &untrusted) {
			t.Errorf("err = %v, want UntrustedIssuerError", err)
		}
	})

	t.Run("requires an issuer callback", func(t *testing.T) {
		if _, err := dynoid.New("svc").VerifyIDToken(ctx, sign(time.Minute)); !errors.Is(err, dynoid.ErrMustCheckIssuer) {
			t.Errorf("err = %v, want ErrMustCheckIssuer", err)
		}
	})
}
