package basicauth

import (
	"os"
	"reflect"
	"testing"

	"github.com/joeshaw/envdecode"
)

func TestCredentialsWithEnvDecode(t *testing.T) {
	os.Setenv("TEST_VALID_CREDS", "user:pass;user2:pass2")
	defer func() {
		os.Setenv("TEST_VALID_CREDS", "")
	}()

	var cfg struct {
		ValidCreds Credentials `env:"TEST_VALID_CREDS,required"`
	}

	if err := envdecode.StrictDecode(&cfg); err != nil {
		t.Fatal(err)
	}

	want := Credentials{{"user", "pass"}, {"user2", "pass2"}}

	if !reflect.DeepEqual(want, cfg.ValidCreds) {
		t.Fatalf("want: %+v, got %+v", want, cfg.ValidCreds)
	}

	// Ensure that we can pass this in to NewChecker all the time, since
	// this is the main.go UX we're going for.
	_ = NewChecker(cfg.ValidCreds)
}

func TestCredentials(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Credentials
		wantErr error
	}{
		{
			name:  "one credential",
			input: "user:pass",
			want:  &Credentials{{"user", "pass"}},
		},
		{
			name:  "two credentials",
			input: "user:pass;user2:pass2",
			want:  &Credentials{{"user", "pass"}, {"user2", "pass2"}},
		},
		{
			name:    "bad format",
			input:   "user",
			want:    &Credentials{},
			wantErr: errMalformedCredentials,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := &Credentials{}
			gotErr := got.Decode(test.input)

			if !reflect.DeepEqual(test.wantErr, gotErr) {
				t.Fatalf("want err: %v, got %v", test.wantErr, gotErr)
			}

			if !reflect.DeepEqual(test.want, got) {
				t.Fatalf("wanted credentials: %+v, got %+v", test.want, got)
			}
		})
	}
}

func TestChecker(t *testing.T) {
	checker := NewChecker([]Credential{
		{Username: "username", Password: "password"},
		{Username: "different", Password: ""},
		{Username: "", Password: "secret"},
	})
	cases := []struct {
		username, password string
		want               bool
	}{
		{"username", "password", true},
		{"invalid", "password", false},
		{"username", "invalid", false},
		{"different", "", true},
		{"", "secret", true},
		{"", "", false},                 // username and password required
		{"different", "invalid", false}, // password match required
		{"invalid", "secret", false},    // username match required
		{"", "password", false},         // username match required
	}
	for i, tt := range cases {
		if got := checker.Valid(tt.username, tt.password); got != tt.want {
			t.Errorf("%d. got %v, want %v", i+1, got, tt.want)
		}
	}
}

// parseCredential returns a Credential instance built from the username and
// password in the provided colon-separated string.
func TestParseCredential(t *testing.T) {
	cred, err := parseCredential("username:password")
	if err != nil {
		t.Fatal(err)
	}
	if cred.Username != "username" && cred.Password != "password" {
		t.Fatalf("got %s:%s but want username:password", cred.Username, cred.Password)
	}
}

// parseCredential allows the password to be blank.
func TestParseCredentialWithMissingPassword(t *testing.T) {
	cred, err := parseCredential("username:")
	if err != nil {
		t.Fatal(err)
	}
	if cred.Username != "username" && cred.Password != "" {
		t.Fatalf("got `%s:%s` but want `username:`", cred.Username, cred.Password)
	}
}

// parseCredential allows the username to be blank.
func TestParseCredentialWithMissingUsername(t *testing.T) {
	cred, err := parseCredential(":secret")
	if err != nil {
		t.Fatal(err)
	}
	if cred.Username != "" && cred.Password != "secret" {
		t.Fatalf("got `%s:%s` but want `:secret`", cred.Username, cred.Password)
	}
}

// parseCredential requires the username or password to be present.
func TestParseCredentialWithMissingValues(t *testing.T) {
	if _, err := parseCredential(":"); err == nil {
		t.Fatal("got nil but want error")
	}
}

// parseCredential returns an error if the specified credentials don't contain
// both a usernamer and password.
func TestParseCredentialWithMalformedCredentials(t *testing.T) {
	if _, err := parseCredential("malformed"); err == nil {
		t.Fatal("got nil but want error")
	}
}
