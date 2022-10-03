package dynoidtest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/golang-jwt/jwt/v4"
	jose "gopkg.in/square/go-jose.v2"
)

type Issuer struct {
	key *rsa.PrivateKey
}

func New() (*Issuer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return &Issuer{key: key}, nil
}

func (iss *Issuer) GenerateIDToken(clientID string) (string, error) {
	now := time.Now()

	claims := &jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings([]string{"heroku"}),
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
	r := chi.NewRouter()

	r.Get("/issuers/test/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
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

	r.Get("/issuers/test/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		jwks := &jose.JSONWebKeySet{}
		jwks.Keys = append(jwks.Keys, jose.JSONWebKey{Key: rt.issuer.key.Public(), KeyID: "primary"})

		header := w.Header()
		header.Set("Content-Type", "application/jwk-set+json")

		w.WriteHeader(http.StatusOK)

		enc := json.NewEncoder(w)
		_ = enc.Encode(jwks)
	})

	rt.handler = r
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.once.Do(rt.init)

	rec := httptest.NewRecorder()

	rt.handler.ServeHTTP(rec, req)

	return rec.Result(), nil
}
