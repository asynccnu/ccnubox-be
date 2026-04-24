package proxy

import (
	"net/http"
	"testing"
)

func TestWithProxyParsesEscapedCredentials(t *testing.T) {
	tr := &HttpTransport{&http.Transport{}}
	WithProxy("http://user%40example.com:p%40ss%3A%2F%3F%23word@127.0.0.1:8080")(tr)

	req, err := http.NewRequest(http.MethodGet, "https://account.ccnu.edu.cn", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("Proxy() error = %v", err)
	}
	if proxyURL == nil {
		t.Fatal("Proxy() returned nil URL")
	}

	username := proxyURL.User.Username()
	password, _ := proxyURL.User.Password()
	if username != "user@example.com" {
		t.Fatalf("username = %q, want %q", username, "user@example.com")
	}
	if password != "p@ss:/?#word" {
		t.Fatalf("password = %q, want %q", password, "p@ss:/?#word")
	}
}
