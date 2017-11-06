package heroku

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/heroku/cedar/lib/grpc/tokenauth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type contextKey string

const (
	apiIDcontextKey    = contextKey("hk-api-auth-id")
	apiEmailContextKey = contextKey("hk-api-auth-email")
)

const (
	apiCredsUsername = "api-username"
	apiCredsToken    = "api-token"
)

var apiBase = "https://api.heroku.com"

// Authorizer is an Authorizer that will validate users against the Heroku API.
// It will tag the users ID in the context for the call.
type Authorizer struct {
	// RequireEmailDomains is a list of email domain name to assert the user is
	// a member of
	RequireEmailDomains []string
	// APIBase is the Heroku API instance that will be contact. Default is prod
	APIBase string
	// Logger wil have errors reported to it
	Logger logrus.FieldLogger
	// HTTPClient to use for requests to the API
	HTTPClient *http.Client
}

// APIUser represents the user object returned from the heroku api at /account
// ref https://devcenter.heroku.com/articles/platform-api-reference#account
type APIUser struct {
	Email    string `json:"email"`
	ID       string `json:"id"`
	Verified bool   `json:"verified"`
}

// Authorize will authorize the RPC call against the Heroku API. The Users's ID
// will be embedded in the context if the authorization is successful.
func (a *Authorizer) Authorize(ctx context.Context, req *tokenauth.RPC, creds map[string]string) (context.Context, error) {
	if req == nil {
		return ctx, grpc.Errorf(codes.Internal, "required argument is nil: request")
	}
	login, ok := creds[apiCredsUsername]
	if !ok {
		return ctx, grpc.Errorf(codes.Unauthenticated, "unable to authorize request without a user login")
	}
	token, ok := creds[apiCredsToken]
	if !ok {
		return ctx, grpc.Errorf(codes.Unauthenticated, "unable to authorize request without a user password")
	}
	if req.Package == "" {
		return ctx, grpc.Errorf(codes.Internal, "required RPC field is nil: Package")
	}
	if req.Service == "" {
		return ctx, grpc.Errorf(codes.Internal, "required RPC field is nil: Service")
	}
	if req.Method == "" {
		return ctx, grpc.Errorf(codes.Internal, "required RPC field is nil: Method")
	}

	// validate creds against API, get email from response
	u, ok, err := a.fetchAPIAccount(login, token)
	if err != nil {
		return ctx, grpc.Errorf(codes.Unknown, "Error validating creds against Heroku API")
	}
	if !ok {
		return ctx, grpc.Errorf(codes.Unauthenticated, "User %s access denied by API", login)
	}

	if !u.Verified {
		return ctx, grpc.Errorf(codes.Unauthenticated, "User %s not verififed", u.Email)
	}

	email := u.Email
	id := u.ID

	// Ensure the email is valid
	valid := false
	for _, m := range a.RequireEmailDomains {
		if strings.HasSuffix(email, "@"+m) {
			valid = true
		}
	}

	if !valid {
		return ctx, grpc.Errorf(codes.PermissionDenied, "User %s is not granted access to this system", email)

	}

	ctx = ContextWithAPIID(ctx, id)
	ctx = ContextWithAPIEmail(ctx, email)

	return ctx, nil
}

// APIIDFromContext will fetch the API User's ID from the context. If it was not found, OK will be false
func APIIDFromContext(ctx context.Context) (id string, ok bool) {
	id, ok = ctx.Value(apiIDcontextKey).(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

// ContextWithAPIID adds the given user ID to the context
func ContextWithAPIID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, apiIDcontextKey, uid)
}

// APIEmailFromContext will fetch the API User's e-mail address from the context.
// If it was not found, OK will be false
func APIEmailFromContext(ctx context.Context) (id string, ok bool) {
	id, ok = ctx.Value(apiEmailContextKey).(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

// ContextWithAPIEmail adds the given user email to the context
func ContextWithAPIEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, apiEmailContextKey, email)
}

// Creds creates a map of credentials suitible for passing with calls from the
// passed username and token.
func Creds(username, token string) map[string]string {
	return map[string]string{
		apiCredsUsername: username,
		apiCredsToken:    token,
	}
}

// fetchAPIAccount will call the Heroku api /account method with the passed in
// creds, returning the user info found, ok of false if user invalid, or the
// error returned
func (a *Authorizer) fetchAPIAccount(login, password string) (user *APIUser, ok bool, err error) {
	ab := a.APIBase
	if ab == "" {
		ab = apiBase
	}
	hc := a.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	req, err := http.NewRequest("GET", ab+"/account", nil)
	req.SetBasicAuth(login, password)
	req.Header.Add("Accept", "application/vnd.heroku+json; version=3")
	resp, err := hc.Do(req)
	if err != nil {
		a.Logger.WithError(err).Error("Error fetching account from API")
		return nil, false, errors.Wrap(err, "Error fetching account from API")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		a.Logger.Errorf("API returned non-200 status code %d", resp.StatusCode)
		return nil, false, fmt.Errorf("API returned non-200 status code %d", resp.StatusCode)
	}

	au := &APIUser{}
	err = json.NewDecoder(resp.Body).Decode(au)
	if err != nil {
		return nil, false, errors.Wrap(err, "Error decoding API response")
	}
	return au, true, nil
}
