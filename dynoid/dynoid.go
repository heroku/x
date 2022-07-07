package dynoid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

type IssuerCallback func(string) error

type Verifier struct {
	IssuerCallback IssuerCallback
	config         *oidc.Config
	providers      sync.Map
}

func New(clientID string) *Verifier {
	return &Verifier{config: &oidc.Config{ClientID: clientID}}
}

func AllowHerokuHost(host string) IssuerCallback {
	return func(issuer string) error {
		if !strings.HasPrefix(issuer, fmt.Sprintf("https://oidc.%v/", host)) {
			return fmt.Errorf("untrusted issuer: %v", issuer)
		}

		return nil
	}
}

func (v *Verifier) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	issuer, err := parseIssuer(rawIDToken)
	if err != nil {
		return nil, err
	}

	if v.IssuerCallback == nil {
		return nil, fmt.Errorf("must check issuer")
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

func (v *Verifier) provider(ctx context.Context, issuer string) (*oidc.Provider, error) {
	provider, ok := v.providers.Load(issuer)
	if ok {
		return provider.(*oidc.Provider), nil
	}

	p, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	provider, _ = v.providers.LoadOrStore(issuer, p)

	return provider.(*oidc.Provider), nil
}

func parseIssuer(p string) (string, error) {
	parts := strings.Split(p, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("malformed token: expected 3 parts got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("unable to decode token: %w", err)
	}

	v := struct {
		Issuer string `json:"iss"`
	}{}

	err = json.Unmarshal(payload, &v)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal token: %w", err)
	}

	return v.Issuer, nil
}
