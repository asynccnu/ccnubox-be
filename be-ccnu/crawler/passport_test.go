package crawler

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestExtractField(t *testing.T) {
	body := `<input name="lt" value="LT-123">`

	got, err := extractField(body, `name="lt".+value="(.+)"`, "lt")
	if err != nil {
		t.Fatalf("extractField() error = %v", err)
	}
	if got != "LT-123" {
		t.Fatalf("unexpected value: %s", got)
	}
}

func TestExtractFieldMissing(t *testing.T) {
	_, err := extractField(`<html></html>`, `name="lt".+value="(.+)"`, "lt")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPassportGetParamsFromHtml(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		headers := http.Header{}
		headers.Add("Set-Cookie", "JSESSIONID=session-1; Path=/")
		body := `
<input name="lt" value="LT-1">
<input name="execution" value="EXE-1">
<input name="_eventId" value="submit">
`
		return newStringResponse(req, http.StatusOK, body, headers), nil
	})

	got, err := NewPassport(client).getParamsFromHtml(context.Background())
	if err != nil {
		t.Fatalf("getParamsFromHtml() error = %v", err)
	}
	if got.lt != "LT-1" || got.execution != "EXE-1" || got._eventId != "submit" || got.JSESSIONID != "session-1" {
		t.Fatalf("unexpected params: %+v", got)
	}
}

func TestPassportGetParamsFromHtmlMissingCookie(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		body := `<input name="lt" value="LT-1">`
		return newStringResponse(req, http.StatusOK, body, nil), nil
	})

	_, err := NewPassport(client).getParamsFromHtml(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPassportLoginCCNUPassport(t *testing.T) {
	params := &accountRequestParams{
		lt:         "LT-1",
		execution:  "EXE-1",
		_eventId:   "submit",
		submit:     "LOGIN",
		JSESSIONID: "session-1",
	}

	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if !strings.Contains(req.URL.String(), ";jsessionid=session-1") {
			t.Fatalf("unexpected url: %s", req.URL.String())
		}

		body := make([]byte, req.ContentLength)
		if _, err := req.Body.Read(body); err != nil && err.Error() != "EOF" {
			t.Fatalf("read body: %v", err)
		}

		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("parse body: %v", err)
		}
		if values.Get("username") != "2024" || values.Get("password") != "secret" {
			t.Fatalf("unexpected credentials: %v", values)
		}

		headers := http.Header{}
		headers.Add("Set-Cookie", "CASTGC=ticket; Path=/")
		return newStringResponse(req, http.StatusOK, "ok", headers), nil
	})

	err := NewPassport(client).loginCCNUPassport(context.Background(), "2024", "secret", params)
	if err != nil {
		t.Fatalf("loginCCNUPassport() error = %v", err)
	}
}

func TestPassportLoginCCNUPassportIncorrectPassword(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		headers := http.Header{}
		headers.Add("Set-Cookie", "CASTGC=ticket; Path=/")
		return newStringResponse(req, http.StatusOK, "您输入的用户名或密码有误", headers), nil
	})

	err := NewPassport(client).loginCCNUPassport(context.Background(), "2024", "secret", &accountRequestParams{JSESSIONID: "s"})
	if err == nil || !strings.Contains(err.Error(), INCorrectPASSWORD.Error()) {
		t.Fatalf("expected incorrect password error, got %v", err)
	}
}

func TestPassportLoginPassport(t *testing.T) {
	var postCount int

	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.String() == LoginCCNUPassPortURL:
			headers := http.Header{}
			headers.Add("Set-Cookie", "JSESSIONID=session-1; Path=/")
			body := `
<input name="lt" value="LT-1">
<input name="execution" value="EXE-1">
<input name="_eventId" value="submit">
`
			return newStringResponse(req, http.StatusOK, body, headers), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.String(), ";jsessionid=session-1"):
			postCount++
			headers := http.Header{}
			headers.Add("Set-Cookie", "CASTGC=ticket; Path=/")
			return newStringResponse(req, http.StatusOK, "ok", headers), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	ok, err := NewPassport(client).LoginPassport(context.Background(), "2024", "secret")
	if err != nil {
		t.Fatalf("LoginPassport() error = %v", err)
	}
	if !ok {
		t.Fatal("expected success")
	}
	if postCount != 1 {
		t.Fatalf("unexpected post count: %d", postCount)
	}
}

func TestPassportLoginPassportIncorrectPassword(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		switch req.Method {
		case http.MethodGet:
			headers := http.Header{}
			headers.Add("Set-Cookie", "JSESSIONID=session-1; Path=/")
			body := `
<input name="lt" value="LT-1">
<input name="execution" value="EXE-1">
<input name="_eventId" value="submit">
`
			return newStringResponse(req, http.StatusOK, body, headers), nil
		case http.MethodPost:
			headers := http.Header{}
			headers.Add("Set-Cookie", "CASTGC=ticket; Path=/")
			return newStringResponse(req, http.StatusOK, "您输入的用户名或密码有误", headers), nil
		default:
			t.Fatalf("unexpected method: %s", req.Method)
			return nil, nil
		}
	})

	ok, err := NewPassport(client).LoginPassport(context.Background(), "2024", "bad")
	if err == nil || !strings.Contains(err.Error(), INCorrectPASSWORD.Error()) {
		t.Fatalf("expected incorrect password error, got %v", err)
	}
	if ok {
		t.Fatal("expected failure")
	}
}
