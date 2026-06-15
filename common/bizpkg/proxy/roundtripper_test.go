package proxy

import (
	"net/url"
	"testing"
)

func TestParseURLPreservesCredentials(t *testing.T) {
	// This test verifies that url.Parse correctly handles already-encoded credentials,
	// which is what be-proxy sends back in its addr/backup fields.
	u, err := url.Parse("http://user%40example.com:p%40ss%3A%2F%3F%23word@127.0.0.1:8080")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	username := u.User.Username()
	password, _ := u.User.Password()
	if username != "user@example.com" {
		t.Fatalf("username = %q, want %q", username, "user@example.com")
	}
	if password != "p@ss:/?#word" {
		t.Fatalf("password = %q, want %q", password, "p@ss:/?#word")
	}
}
