// Package cleanurl provides utilities for scrubbing basic auth credentials
// from URLs.
package cleanurl

import "net/url"

// ExtractCredentials extracts and scrubs basic auth credentials from a URL to
// ensure that they never get logged.
func ExtractCredentials(uri *url.URL) (*url.URL, string, string) {
	cleanURL, _ := url.Parse(uri.String())
	username := ""
	password := ""
	if cleanURL.User != nil {
		username = uri.User.Username()
		password, _ = uri.User.Password()
	}
	cleanURL.User = nil
	return cleanURL, username, password
}
