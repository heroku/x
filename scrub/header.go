package scrub

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	scrubbedValue       = "[SCRUBBED]"
	authHeaderLowerCase = "authorization"
)

var (
	// copy from https://github.com/heroku/rollbar-blanket/blob/master/lib/rollbar/blanket/headers.rb
	restrictedHeaders = map[string]bool{
		"cookie":                      true,
		"heroku-authorization-token":  true,
		"heroku-two-factor-code":      true,
		"heroku-umbrella-token":       true,
		"http_authorization":          true,
		"http_heroku_two_factor_code": true,
		"http_x_csrf_token":           true,
		"oauth-access-token":          true,
		"omniauth.auth":               true,
		"set-cookie":                  true,
		"x-csrf-token":                true,
		"x_csrf_token":                true,
		"authorization":               true,
	}
)

// Header removes a subset of "sensitive" HTTP headers as used by parts of Heroku's
// conventions for API design. The output of this function is safe to be logged
// except in high-security scenarios.
func Header(h http.Header) http.Header {
	scrubbedHeader := http.Header{}
	for k, v := range h {
		if strings.ToLower(k) == authHeaderLowerCase {
			scrubbedValues := []string{}
			for _, auth := range v {
				substrs := strings.SplitN(auth, " ", 2)
				scrubbed := scrubbedValue
				if len(substrs) > 1 {
					scrubbed = fmt.Sprintf("%s %s", substrs[0], scrubbedValue)
				}
				scrubbedValues = append(scrubbedValues, scrubbed)
			}
			scrubbedHeader[k] = scrubbedValues
		} else if _, contains := restrictedHeaders[strings.ToLower(k)]; contains {
			scrubbedHeader[k] = []string{scrubbedValue}
		} else {
			scrubbedHeader[k] = v
		}
	}

	return scrubbedHeader
}
