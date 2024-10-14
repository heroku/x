package dynoid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	defaultAudience = "heroku"

	//nolint: gosec
	defaultTokenPath = "/etc/heroku/dyno_id_token"
)

// Returned by an IssuerCallback getting an issuer it doesn't trust
type UntrustedIssuerError struct {
	Issuer string
}

func (e *UntrustedIssuerError) Error() string {
	return fmt.Sprintf("untrusted issuer: %v", e.Issuer)
}

// Returned when the token doesn't match the expected format
type MalformedTokenError struct {
	err error
}

func (e *MalformedTokenError) Error() string {
	return fmt.Sprintf("malformed token: %s", e.err.Error())
}

func (e *MalformedTokenError) Unwrap() error {
	return e.err
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

// Subject contains information about the app and dyno the token was issued for
type Subject struct {
	AppID   string `json:"app_id"`
	AppName string `json:"app_name"`
	Dyno    string `json:"dyno"`
}

func (s *Subject) LogValue() slog.Value {
	if s == nil {
		return (&Subject{}).LogValue()
	}

	return slog.GroupValue(
		slog.String("app_id", s.AppID),
		slog.String("app_name", s.AppName),
		slog.String("dyno", s.Dyno),
	)
}

func (s *Subject) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Subject) UnmarshalText(text []byte) error {
	if s == nil {
		return fmt.Errorf("cannot unmarshal to a nil pointer")
	}

	sub := string(text)
	parts := strings.Split(sub, ":")
	if len(parts) != 5 || parts[0] != "app" || parts[3] != "dyno" {
		return fmt.Errorf("unexpected subject format: %q", sub)
	}

	app := strings.Split(parts[1], ".")
	if len(app) != 2 {
		return fmt.Errorf("unexpected subject format: %q", sub)
	}

	s.AppID = app[0]
	s.AppName = app[1]
	s.Dyno = parts[4]

	return nil
}

func (s *Subject) String() string {
	if s == nil {
		return ""
	}

	return fmt.Sprintf("app:%s.%s::dyno:%s", s.AppID, s.AppName, s.Dyno)
}

// Token contains all of the token information stored by Heroku when it's issued
type Token struct {
	IDToken *oidc.IDToken `json:"-"`
	SpaceID string        `json:"space_id"`
	Subject *Subject      `json:"subject"`
}

func (t *Token) LogValue() slog.Value {
	if t == nil {
		return (&Token{}).LogValue()
	}

	return slog.GroupValue(
		slog.String("space_id", t.SpaceID),
		slog.Any("subject", t.Subject),
	)
}

// LocalTokenPath returns the path on disk to the token for the given audience
func LocalTokenPath(audience string) string {
	if audience == defaultAudience {
		return defaultTokenPath
	}

	return fmt.Sprintf("/etc/heroku/dyno-id/%s/token", audience)
}

type osReader struct{}

func (*osReader) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (*osReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// DefaultFS is used by [ReadLocal] and [ReadLocalToken] to retrieve tokens.
//
// By default they are retrieved via [os.Open] and [os.ReadFile].
//
// This is useful when testing code that uses DynoID.
var DefaultFS fs.ReadFileFS = &osReader{}

// ReadLocal reads the local machines token for the given audience
//
// Suitable for passing as a bearer token
func ReadLocal(audience string) (string, error) {
	rawToken, err := DefaultFS.ReadFile(LocalTokenPath(audience))
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

	verifier := NewWithCallback(audience, func(string) error { return nil })

	return verifier.Verify(ctx, rawToken)
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
func (v *Verifier) Verify(ctx context.Context, rawIDToken string) (*Token, error) {
	if v == nil {
		return New("").Verify(ctx, rawIDToken)
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

	token, err := verifier.Verify(ctx, rawIDToken)
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
			err: fmt.Errorf("expected 3 parts got %d", len(parts)),
		}
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", &MalformedTokenError{
			err: fmt.Errorf("unable to decode token: %w", err),
		}
	}

	v := struct {
		Issuer string `json:"iss"`
	}{}

	err = json.Unmarshal(payload, &v)
	if err != nil {
		return "", &MalformedTokenError{
			err: fmt.Errorf("unable to unmarshal token: %w", err),
		}
	}

	return v.Issuer, nil
}
