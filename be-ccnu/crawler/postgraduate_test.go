package crawler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"
)

func TestParseRSAPublicKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	modulus := base64.StdEncoding.EncodeToString(key.PublicKey.N.Bytes())
	exponent := base64.StdEncoding.EncodeToString([]byte{0x01, 0x00, 0x01})

	pub, err := parseRSAPublicKey(modulus, exponent)
	if err != nil {
		t.Fatalf("parseRSAPublicKey() error = %v", err)
	}
	if pub.N.Cmp(key.PublicKey.N) != 0 || pub.E != key.PublicKey.E {
		t.Fatalf("unexpected public key: %+v", pub)
	}
}

func TestParseRSAPublicKeyInvalidBase64(t *testing.T) {
	_, err := parseRSAPublicKey("bad", "AQAB")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEncryptPasswordJSStyle(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	encrypted, err := encryptPasswordJSStyle("secret", &key.PublicKey)
	if err != nil {
		t.Fatalf("encryptPasswordJSStyle() error = %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}

	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, key, raw)
	if err != nil {
		t.Fatalf("DecryptPKCS1v15() error = %v", err)
	}
	if string(decrypted) != "secret" {
		t.Fatalf("unexpected decrypted value: %s", string(decrypted))
	}
}

func TestPostGraduateFetchPublicKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	modulus := base64.StdEncoding.EncodeToString(key.PublicKey.N.Bytes())
	exponent := base64.StdEncoding.EncodeToString([]byte{0x01, 0x00, 0x01})
	body := `{"modulus":"` + modulus + `","exponent":"` + exponent + `"}`

	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return newStringResponse(req, http.StatusOK, body, nil), nil
	})

	got, err := NewPostGraduate(client).FetchPublicKey(context.Background())
	if err != nil {
		t.Fatalf("FetchPublicKey() error = %v", err)
	}
	if got.N.Cmp(key.PublicKey.N) != 0 || got.E != key.PublicKey.E {
		t.Fatalf("unexpected public key: %+v", got)
	}
}

func TestPostGraduateLoginPostgraduateSystemIncorrectPassword(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return newStringResponse(req, http.StatusOK, "用户名或密码不正确", nil), nil
	})

	err = NewPostGraduate(client).LoginPostgraduateSystem(context.Background(), "2024", "bad", &key.PublicKey)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPostGraduateGetCookie(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	var client *http.Client
	client = newTestClient(t, func(req *http.Request) (*http.Response, error) {
		rootURL, err := url.Parse("https://grd.ccnu.edu.cn/yjsxt")
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		client.Jar.SetCookies(rootURL, []*http.Cookie{
			{Name: "JSESSIONID", Value: "session-1"},
			{Name: "route", Value: "node-1"},
		})
		return newStringResponse(req, http.StatusOK, "ok", nil), nil
	})

	got, err := NewPostGraduate(client).GetCookie(context.Background(), "2024", "secret", &key.PublicKey)
	if err != nil {
		t.Fatalf("GetCookie() error = %v", err)
	}
	if got != "JSESSIONID=session-1;route=node-1" {
		t.Fatalf("unexpected cookie string: %s", got)
	}
}

func TestPostGraduateGetCookieMissingCookie(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	client := newTestClient(t, func(req *http.Request) (*http.Response, error) {
		return newStringResponse(req, http.StatusOK, "ok", nil), nil
	})

	_, err = NewPostGraduate(client).GetCookie(context.Background(), "2024", "secret", &key.PublicKey)
	if err == nil {
		t.Fatal("expected error")
	}
}
