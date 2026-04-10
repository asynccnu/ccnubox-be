package crawler

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestLibraryLoginLibrary(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.String() != PG_URL_LOGIN_LIBRARY {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}
		return newStringResponse(req, http.StatusOK, "", nil), nil
	})

	err := NewLibrary(client, "secret").LoginLibrary(context.Background())
	if err != nil {
		t.Fatalf("LoginLibrary() error = %v", err)
	}
}

func TestLibraryGetSeatAuthTokenFromLibrary(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.String(), PG_URL_SEAT_LIBRARY_PREFIX):
			redirectReq := req.Clone(req.Context())
			redirectReq.URL, _ = url.Parse(PG_URL_SEAT_LIBRARY_PREFIX + "#/entry?token=raw-token")
			return newStringResponse(redirectReq, http.StatusOK, "", nil), nil
		case req.Method == http.MethodPost && req.URL.String() == PG_URL_SEAT_AUTH_TOKEN_LIBRARY_PREFIX+"raw-token":
			return newStringResponse(req, http.StatusOK, `{"status":true,"code":200,"data":{"token":"seat-auth"}}`, nil), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	got, err := NewLibrary(client, "secret").GetSeatAuthTokenFromLibrary(context.Background())
	if err != nil {
		t.Fatalf("GetSeatAuthTokenFromLibrary() error = %v", err)
	}
	if got != "seat-auth" {
		t.Fatalf("unexpected token: %s", got)
	}
}

func TestLibraryCheckLibrarySeatToken(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Token") != "seat-auth" {
			t.Fatalf("unexpected token header: %s", req.Header.Get("Token"))
		}
		if req.Header.Get("LoginType") != "PC" {
			t.Fatalf("unexpected login type: %s", req.Header.Get("LoginType"))
		}
		if req.Header.Get("X-Hmac-Request-Key") == "" || req.Header.Get("X-Request-Date") == "" || req.Header.Get("X-Request-Id") == "" {
			t.Fatal("expected signed headers")
		}
		return newStringResponse(req, http.StatusOK, "", nil), nil
	})

	ok, err := NewLibrary(client, "secret").CheckLibrarySeatToken(context.Background(), "seat-auth")
	if err != nil {
		t.Fatalf("CheckLibrarySeatToken() error = %v", err)
	}
	if !ok {
		t.Fatal("expected token to be valid")
	}
}

func TestLibraryGetDiscussionAuthTokenFromLibrary(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.String(), PG_URL_DISCUSSION_LIBRARY_PREFIX):
			redirectReq := req.Clone(req.Context())
			redirectReq.URL, _ = url.Parse(PG_URL_DISCUSSION_LIBRARY_PREFIX + "#/entry?token=raw-discussion")
			return newStringResponse(redirectReq, http.StatusOK, "", nil), nil
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.String(), PG_URL_DISCUSSION_AUTH_TOKEN_LIBRARY_PREFIX):
			if req.URL.Query().Get("token") != "raw-discussion" {
				t.Fatalf("unexpected raw token: %s", req.URL.Query().Get("token"))
			}
			return newStringResponse(req, http.StatusOK, `{"data":{"token":"discussion-auth"}}`, nil), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	got, err := NewLibrary(client, "secret").GetDiscussionAuthTokenFromLibrary(context.Background())
	if err != nil {
		t.Fatalf("GetDiscussionAuthTokenFromLibrary() error = %v", err)
	}
	if got != "discussion-auth" {
		t.Fatalf("unexpected token: %s", got)
	}
}

func TestLibraryCheckLibraryDiscussionToken(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "discussion-auth" {
			t.Fatalf("unexpected authorization header: %s", req.Header.Get("Authorization"))
		}
		return newStringResponse(req, http.StatusOK, "", nil), nil
	})

	ok, err := NewLibrary(client, "secret").CheckLibraryDiscussionToken(context.Background(), "discussion-auth")
	if err != nil {
		t.Fatalf("CheckLibraryDiscussionToken() error = %v", err)
	}
	if !ok {
		t.Fatal("expected token to be valid")
	}
}

func TestLibraryParseRawTokenFromUrl(t *testing.T) {
	got, err := NewLibrary(nil, "").parseRawTokenFromUrl("https://example.com#/entry?token=abc")
	if err != nil {
		t.Fatalf("parseRawTokenFromUrl() error = %v", err)
	}
	if got != "abc" {
		t.Fatalf("unexpected token: %s", got)
	}
}

func TestLibraryBuildLibraryTokenUrl(t *testing.T) {
	got := NewLibrary(nil, "").buildLibraryTokenUrl("https://example.com/path")
	if !strings.HasPrefix(got, "https://example.com/path?rand=") {
		t.Fatalf("unexpected url: %s", got)
	}
}
