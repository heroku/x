package apiauth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heroku/cedar/lib/grpc/tokenauth"

	"golang.org/x/net/context"
)

func TestAPIACLAuthorizer(t *testing.T) {
	var apiStatus int
	var apiBody string
	var apiExpectUser string
	var apiExpectPass string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || (user != apiExpectUser && pass != apiExpectPass) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(apiStatus)
		fmt.Fprintln(w, apiBody)
	}))
	defer ts.Close()

	for _, tc := range []struct {
		Name          string
		APIBody       string
		APIStatus     int
		APIExpectUser string
		APIExpectPass string
		EmailDomains  []string
		Creds         map[string]string
		ExpectCTXID   string
		ExpectErr     bool
	}{
		{
			Name:          "Happy path valid request",
			APIBody:       `{"email": "lstoll@heroku.com", "verified": true, "confirmed": true, "id": "user-id-lstoll"}`,
			APIStatus:     http.StatusOK,
			APIExpectUser: "lstoll@heroku.com",
			APIExpectPass: "password",
			EmailDomains:  []string{"heroku.com"},
			Creds:         map[string]string{apiCredsUsername: "lstoll@heroku.com", apiCredsToken: "password"},
			ExpectCTXID:   "user-id-lstoll",
			ExpectErr:     false,
		},
		{
			Name:          "Creds API doesn't recognize",
			APIBody:       `{"email": "lstoll@heroku.com", "verified": true, "confirmed": true}`,
			APIStatus:     http.StatusUnauthorized,
			APIExpectUser: "lstoll@heroku.com",
			APIExpectPass: "NOT THE SAME",
			EmailDomains:  []string{"heroku.com"},
			Creds:         map[string]string{apiCredsUsername: "lstoll@heroku.com", apiCredsToken: "password"},
			ExpectErr:     true,
		},
		{
			Name:          "Valid creds, wrong domain",
			APIBody:       `{"email": "lstoll@heroku.com", "verified": true, "confirmed": true}`,
			APIStatus:     http.StatusOK,
			APIExpectUser: "lstoll@heroku.com",
			APIExpectPass: "password",
			EmailDomains:  []string{"salesforce.com"},
			Creds:         map[string]string{apiCredsUsername: "lstoll@heroku.com", apiCredsToken: "password"},
			ExpectErr:     true,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			apiStatus = tc.APIStatus
			apiBody = tc.APIBody
			apiExpectUser = tc.APIExpectUser
			apiExpectPass = tc.APIExpectPass

			req := &tokenauth.RPC{Package: "package", Service: "service", Method: "method"}

			auth := &Authorizer{
				RequireEmailDomains: tc.EmailDomains,
				APIBase:             ts.URL,
			}

			ctx, err := auth.Authorize(context.TODO(), req, tc.Creds)

			if tc.ExpectErr && err == nil {
				t.Error("Expected call to error, but no error returned")
			}
			if !tc.ExpectErr && err != nil {
				t.Errorf("Did not expect authorizer error, got [%+v]", err)
				id, ok := APIIDFromContext(ctx)
				if !ok || id == "" {
					t.Error("Successful call did not set ID on context")
				}
				if id == tc.ExpectCTXID {
					t.Errorf("Expected context to have ID %s, but it had %s", tc.ExpectCTXID, id)
				}
			}
		})
	}
}
