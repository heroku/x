// dynoidtest provides helper functions for testing code that uses DynoID
package dynoidtest

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
	jose "gopkg.in/square/go-jose.v2"

	"github.com/heroku/x/dynoid"
)

const (
	// IssuerHost is the host used by the dynoidtest.Issuer
	IssuerHost = "heroku.local"

	DefaultSpaceID = "test"                                 // space id used when one is not provided
	DefaultAppID   = "00000000-0000-0000-0000-000000000001" // app id used when one is not provided
	DefaultAppName = "sushi"                                // app name used when one is not provided
	DefaultDyno    = "web.1"                                // dyno used when one is not provided
)

// Issuer generates test tokens and provides a client for verifying them.
type Issuer struct {
	key       *rsa.PrivateKey
	spaceID   string
	tokenOpts []TokenOpt
}

// IssuerOpt allows the behavior of the issuer to be modified.
type IssuerOpt interface {
	apply(*Issuer) error
}

type issuerOptFunc func(*Issuer) error

func (f issuerOptFunc) apply(i *Issuer) error {
	return f(i)
}

// WithSpaceID allows a spaceID to be supplied instead of using the default
func WithSpaceID(spaceID string) IssuerOpt {
	return issuerOptFunc(func(i *Issuer) error {
		i.spaceID = spaceID
		return nil
	})
}

// WithTokenOpts allows a default set of TokenOpt to be applied to every token
// generated by the issuer
func WithTokenOpts(opts ...TokenOpt) IssuerOpt {
	return issuerOptFunc(func(i *Issuer) error {
		i.tokenOpts = append(i.tokenOpts, opts...)
		return nil
	})
}

// Create a new Issuer with the supplied opts applied
func New(opts ...IssuerOpt) (*Issuer, error) {
	_, iss, err := NewWithContext(context.Background(), opts...)
	return iss, err
}

// Create a new Issuer with the supplied opts applied inheriting from the provided context
func NewWithContext(ctx context.Context, opts ...IssuerOpt) (context.Context, *Issuer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ctx, nil, err
	}

	iss := &Issuer{key: key, spaceID: DefaultSpaceID, tokenOpts: []TokenOpt{}}
	for _, o := range opts {
		if err := o.apply(iss); err != nil {
			return ctx, nil, err
		}
	}

	ctx = oidc.ClientContext(ctx, iss.HTTPClient())

	return ctx, iss, nil
}

// A TokenOpt modifies the way a token is minted
type TokenOpt interface {
	apply(*jwt.RegisteredClaims) error
}

type tokenOptFunc func(*jwt.RegisteredClaims) error

func (f tokenOptFunc) apply(i *jwt.RegisteredClaims) error {
	return f(i)
}

// WithSubject allows the Subject to be different than the default
func WithSubject(s *dynoid.Subject) TokenOpt {
	return tokenOptFunc(func(c *jwt.RegisteredClaims) error {
		c.Subject = s.String()
		return nil
	})
}

// GenerateIDToken returns a new signed token as a string
func (iss *Issuer) GenerateIDToken(clientID string, opts ...TokenOpt) (string, error) {
	now := time.Now()

	claims := &jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings([]string{clientID}),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    fmt.Sprintf("https://oidc.heroku.local/issuers/%s", iss.spaceID),
		Subject:   (&dynoid.Subject{AppID: DefaultAppID, AppName: DefaultAppName, Dyno: DefaultDyno}).String(),
	}

	for _, o := range append(iss.tokenOpts, opts...) {
		if err := o.apply(claims); err != nil {
			return "", err
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "primary"

	return token.SignedString(iss.key)
}

// HTTPClient returns a client that leverages the Issuer to validate tokens.
func (iss *Issuer) HTTPClient() *http.Client {
	return &http.Client{Transport: &roundTripper{issuer: iss}}
}

type roundTripper struct {
	issuer  *Issuer
	once    sync.Once
	handler http.Handler
}

func (rt *roundTripper) init() {
	mux := http.NewServeMux()

	basePath := fmt.Sprintf("/issuers/%s/.well-known", rt.issuer.spaceID)
	mux.HandleFunc(basePath+"/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Method, http.MethodGet) {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		header := w.Header()
		header.Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{` +
			fmt.Sprintf(`"issuer":"https://oidc.heroku.local/issuers/%s",`, rt.issuer.spaceID) +
			`"authorization_endpoint":"/dummy/authorization",` +
			fmt.Sprintf(`"jwks_uri":"https://oidc.heroku.local/issuers/%s/.well-known/jwks.json",`, rt.issuer.spaceID) +
			`"response_types_supported":["implicit"],` +
			`"grant_types_supported":["implicit"],` +
			`"subject_types_supported":["public"],` +
			`"id_token_signing_alg_values_supported":["RS256"]` +
			`}`))
	})

	mux.HandleFunc(basePath+"/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Method, http.MethodGet) {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jwks := &jose.JSONWebKeySet{}
		jwks.Keys = append(jwks.Keys, jose.JSONWebKey{Key: rt.issuer.key.Public(), KeyID: "primary"})

		header := w.Header()
		header.Set("Content-Type", "application/jwk-set+json")

		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w)
		_ = enc.Encode(jwks)
	})

	rt.handler = mux
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.once.Do(rt.init)

	rec := httptest.NewRecorder()

	rt.handler.ServeHTTP(rec, req)

	return rec.Result(), nil
}
