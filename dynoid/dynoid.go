package dynoid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	AudienceHeroku = "heroku"
)

// Returned by an IssuerCallback get's an issuer it doesn't trust
type UntrustedIssuerError struct {
	Issuer string
}

func (e *UntrustedIssuerError) Error() string {
	return fmt.Sprintf("untrusted issuer: %v", e.Issuer)
}

// Returned when the token doesn't match the expected format
type MalformedTokenError struct {
	Err error
}

func (e *MalformedTokenError) Error() string {
	return fmt.Sprintf("malformed token: %s", e.Err.Error())
}

func (e *MalformedTokenError) Unwrap() error {
	return e.Err
}

type staticError string

func (e staticError) Error() string {
	return string(e)
}

const (
	ErrMustCheckIssuer staticError = "must check issuer"
)

// An IssuerCallback is called whenever a token is verified to ensure it matches
// some expected criteria.
type IssuerCallback func(issuer string) error

// AllowHerokuHost verifies that the issuer is from Heroku for the given host
// domain
func AllowHerokuHost(host string) IssuerCallback {
	return func(issuer string) error {
		if !strings.HasPrefix(issuer, fmt.Sprintf("https://oidc.%v/", host)) {
			return &UntrustedIssuerError{Issuer: issuer}
		}

		return nil
	}
}

// Subject contains all of the subject information stored by Heroku when issuing
// a token.
type Subject struct {
	AppID      string `json:"app_id"`
	AppName    string `json:"app_name"`
	DynoName   string `json:"dyno_name"`
	DynoNumber int    `json:"dyno_number"`
}

func (s *Subject) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("app_id", s.AppID),
		slog.String("app_name", s.AppName),
		slog.String("dyno", fmt.Sprintf("%s.%d", s.DynoName, s.DynoNumber)),
	)
}

func (s *Subject) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Subject) UnmarshalText(text []byte) error {
	if s == nil {
		*s = Subject{}
	}

	sub := string(text)
	parts := strings.Split(sub, ":")
	if len(parts) != 5 || parts[0] != "app" || parts[3] != "dyno" {
		return fmt.Errorf("unexpected subject format: %q", sub)
	}

	app := strings.Split(parts[1], ".")
	dyno := strings.Split(parts[4], ".")

	if len(app) != 2 || len(dyno) != 2 {
		return fmt.Errorf("unexpected subject format: %q", sub)
	}

	s.AppID = app[0]
	s.AppName = app[1]
	s.DynoName = dyno[0]
	s.DynoNumber, _ = strconv.Atoi(dyno[1])

	if s.DynoNumber == 0 {
		return fmt.Errorf("unexpected subject format: %q", sub)
	}

	return nil
}

func (s *Subject) String() string {
	if s == nil {
		return ""
	}

	return fmt.Sprintf("app:%s.%s::dyno:%s.%d", s.AppID, s.AppName, s.DynoName, s.DynoNumber)
}

// Subject contains all of the token information stored by Heroku when issuing
// a token.
type Token struct {
	IDToken *oidc.IDToken `json:"-"`
	SpaceID string        `json:"space_id"`
	Subject *Subject      `json:"subject"`
}

func (t *Token) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("space_id", t.SpaceID),
		slog.Any("subject", t.Subject),
	)
}

// ReadLocal reads the local machines token for the given audience
//
// Suitable for passing as a bearer token
func ReadLocal(audience string) (string, error) {
	tokenPath := "/etc/heroku/dyno_id_token"

	if audience != "heroku" {
		tokenPath = fmt.Sprintf("/etc/heroku/dyno-id/%s/token", audience)
	}

	rawToken, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", err
	}

	return string(rawToken), nil
}

// ReadLocalToken reads the local machines token for the given audience and
// parses it
func ReadLocalToken(ctx context.Context, audience string) (*Token, error) {
	rawToken, err := ReadLocal(audience)
	if err != nil {
		return nil, fmt.Errorf("failed to read token (%w)", err)
	}

	verifier := NewWithCallback(audience, func(issuer string) error { return nil })

	return verifier.VerifyHeroku(ctx, rawToken)
}

// AllowHerokuSpace verifies that the issuer is from Heroku for the given host
// and space id.
func AllowHerokuSpace(host string, spaceIDs ...string) IssuerCallback {
	return func(issuer string) error {
		for _, id := range spaceIDs {
			if iss := fmt.Sprintf("https://oidc.%s/spaces/%s", host, id); iss == issuer {
				return nil
			}
		}

		return &UntrustedIssuerError{Issuer: issuer}
	}
}

// A Verifier verifies a raw token with it's oids issuer and uses the
// IssuerCallback to ensure it's from a trusted source.
type Verifier struct {
	IssuerCallback IssuerCallback

	config *oidc.Config

	mu        *sync.RWMutex
	providers map[string]*oidc.Provider
}

// Instantiate a new Verifier without an IssuerCallback set.
//
// The IssuerCallback must be set before calling Verify or an error will be
// returned.
func New(clientID string) *Verifier {
	return &Verifier{
		config:    &oidc.Config{ClientID: clientID},
		mu:        &sync.RWMutex{},
		providers: make(map[string]*oidc.Provider),
	}
}

// Instantiate a new Verifier with the IssuerCallback set.
func NewWithCallback(clientID string, callback IssuerCallback) *Verifier {
	v := New(clientID)
	v.IssuerCallback = callback
	return v
}

// Verify validates the given token with the OIDC provider and validates it
// against the IssuerCallback
func (v *Verifier) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	if v == nil {
		*v = *New("")
	}

	if v.IssuerCallback == nil {
		return nil, ErrMustCheckIssuer
	}

	issuer, err := parseIssuer(rawIDToken)
	if err != nil {
		return nil, err
	}

	if err = v.IssuerCallback(issuer); err != nil {
		return nil, err
	}

	provider, err := v.provider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(v.config)

	return verifier.Verify(ctx, rawIDToken)
}

// VerifyHeroku verifies the token and parses the returned issuer and subject
// according to Heroku expected values
func (v *Verifier) VerifyHeroku(ctx context.Context, rawIDToken string) (*Token, error) {
	token, err := v.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token (%w)", err)
	}

	var s Subject
	if err := s.UnmarshalText([]byte(token.Subject)); err != nil {
		return nil, fmt.Errorf("failed to parse subject (%w)", err)
	}

	return &Token{
		IDToken: token,
		SpaceID: path.Base(token.Issuer),
		Subject: &s,
	}, nil
}

func (v *Verifier) provider(ctx context.Context, issuer string) (*oidc.Provider, error) {
	v.mu.RLock()
	if provider, ok := v.providers[issuer]; ok {
		v.mu.RUnlock()
		return provider, nil
	}

	v.mu.RUnlock()
	v.mu.Lock()
	defer v.mu.Unlock()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	v.providers[issuer] = provider

	return provider, nil
}

func parseIssuer(p string) (string, error) {
	parts := strings.Split(p, ".")
	if len(parts) != 3 {
		return "", &MalformedTokenError{
			Err: fmt.Errorf("expected 3 parts got %d", len(parts)),
		}
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", &MalformedTokenError{
			Err: fmt.Errorf("unable to decode token: %w", err),
		}
	}

	v := struct {
		Issuer string `json:"iss"`
	}{}

	err = json.Unmarshal(payload, &v)
	if err != nil {
		return "", &MalformedTokenError{
			Err: fmt.Errorf("unable to unmarshal token: %w", err),
		}
	}

	return v.Issuer, nil
}
