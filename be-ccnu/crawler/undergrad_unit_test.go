package crawler

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestUnderGradLoginUnderGradSystem(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.String() != CASURL {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}

		return newStringResponse(req, http.StatusOK, "", nil), nil
	})

	err := NewUnderGrad(client).LoginUnderGradSystem(context.Background())
	if err != nil {
		t.Fatalf("LoginUnderGradSystem() error = %v", err)
	}
}

func TestUnderGradLoginUnderGradSystemUnexpectedStatus(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return newStringResponse(req, http.StatusBadGateway, "", nil), nil
	})

	err := NewUnderGrad(client).LoginUnderGradSystem(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnderGradGetCookieFromUnderGradSystem(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		t.Fatal("unexpected outbound request")
		return nil, nil
	})

	u, err := url.Parse(pgUrl)
	if err != nil {
		t.Fatalf("parse pgUrl: %v", err)
	}

	client.Jar.SetCookies(u, []*http.Cookie{
		{Name: "JSESSIONID", Value: "abc"},
		{Name: "route", Value: "node1"},
	})

	got, err := NewUnderGrad(client).GetCookieFromUnderGradSystem()
	if err != nil {
		t.Fatalf("GetCookieFromUnderGradSystem() error = %v", err)
	}
	if got != "JSESSIONID=abc; route=node1" {
		t.Fatalf("unexpected cookie string: %s", got)
	}
}

func TestUnderGradGetCookieFromUnderGradSystemNoCookie(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		t.Fatal("unexpected outbound request")
		return nil, nil
	})

	_, err := NewUnderGrad(client).GetCookieFromUnderGradSystem()
	if err == nil {
		t.Fatal("expected error")
	}
}
