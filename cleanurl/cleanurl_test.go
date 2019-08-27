package cleanurl

import (
	"net/url"
	"testing"
)

// ExtractCredentials scrubs basic auth credentials from a URL.
func TestExtractCredentials(t *testing.T) {
	uri, _ := url.Parse("https://buddy:secret@example.com")
	cleanURL, username, password := ExtractCredentials(uri)
	if cleanURL.User != nil {
		t.Fatalf("got %v but want nil", cleanURL.User)
	}
	if username != "buddy" {
		t.Fatalf("got %s but want buddy", username)
	}
	if password != "secret" {
		t.Fatalf("got %s but want secret", password)
	}
}

func TestExtractAPIKey(t *testing.T) {
	uri, _ := url.Parse("https://secret@example.com")
	cleanURL, username, _ := ExtractCredentials(uri)
	if cleanURL.User != nil {
		t.Fatalf("got %v but want nil", cleanURL.User)
	}
	if username != "secret" {
		t.Fatalf("got %s but want secret", username)
	}
}
