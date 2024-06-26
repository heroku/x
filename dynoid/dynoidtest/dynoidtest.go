package dynoidtest

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
	jose "gopkg.in/square/go-jose.v2"
)

const (
	Audience   = "heroku"
	IssuerHost = "heroku.local"
)

type Issuer struct {
	key *rsa.PrivateKey
}

func New() (*Issuer, error) {
	_, iss, err := NewWithContext(context.Background())
	return iss, err
}

func NewWithContext(ctx context.Context) (context.Context, *Issuer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ctx, nil, err
	}

	iss := &Issuer{key: key}
	ctx = oidc.ClientContext(ctx, iss.HTTPClient())

	return ctx, iss, nil
}

func (iss *Issuer) GenerateIDToken(clientID string) (string, error) {
	now := time.Now()

	claims := &jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings([]string{clientID}),
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    "https://oidc.heroku.local/issuers/test",
		Subject:   "app:00000000-0000-0000-0000-000000000001.sushi::dyno:web.1",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "primary"

	return token.SignedString(iss.key)
}

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

	mux.HandleFunc("/issuers/test/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Method, http.MethodGet) {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		header := w.Header()
		header.Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{` +
			`"issuer":"https://oidc.heroku.local/issuers/test",` +
			`"authorization_endpoint":"/dummy/authorization",` +
			`"jwks_uri":"https://oidc.heroku.local/issuers/test/.well-known/jwks.json",` +
			`"response_types_supported":["implicit"],` +
			`"grant_types_supported":["implicit"],` +
			`"subject_types_supported":["public"],` +
			`"id_token_signing_alg_values_supported":["RS256"]` +
			`}`))
	})

	mux.HandleFunc("/issuers/test/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
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
