package crawler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
)

// 随便写的,比较随意
func Test_GetCookie(t *testing.T) {
	studentID, password := integrationCredentials(t)
	p := NewPassport(NewCrawlerClient(proxy.NewDirectHttpProxy(nil), 10*time.Second))
	ctx := context.Background()
	_, err := p.LoginPassport(ctx, studentID, password)
	if err != nil {
		t.Fatal(err)
	}

	ug := NewUnderGrad(p.Client)
	err = ug.LoginUnderGradSystem(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cookie, err := ug.GetCookieFromUnderGradSystem()
	if err != nil {
		t.Fatal(err)
	}
	if cookie == "" {
		t.Fatal("empty undergraduate system cookie")
	}
}

func Test_GetLibraryCookie(t *testing.T) {
	studentID, password := integrationCredentials(t)
	p := NewPassport(NewCrawlerClient(proxy.NewDirectHttpProxy(nil), 10*time.Second))
	ctx := context.Background()
	_, err := p.LoginPassport(ctx, studentID, password)
	if err != nil {
		t.Fatal(err)
	}

	l := NewLibrary(p.Client, "")
	err = l.LoginLibrary(ctx)
	if err != nil {
		t.Fatal(err)
	}

	discussionToken, err := l.GetDiscussionAuthTokenFromLibrary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if discussionToken == "" {
		t.Fatal("empty discussion token")
	}

	valid, err := l.CheckLibraryDiscussionToken(ctx, discussionToken)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("discussion token is invalid")
	}
}

func integrationCredentials(t *testing.T) (string, string) {
	t.Helper()
	studentID := os.Getenv("CCNU_TEST_STUDENT_ID")
	password := os.Getenv("CCNU_TEST_PASSWORD")
	if studentID == "" || password == "" {
		t.Skip("set CCNU_TEST_STUDENT_ID and CCNU_TEST_PASSWORD to run integration tests")
	}
	return studentID, password
}

type cookieRoundTripper func(*http.Request) (*http.Response, error)

func (f cookieRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type cookieResponseBody struct {
	io.Reader
	closed bool
}

func (b *cookieResponseBody) Close() error { b.closed = true; return nil }

func TestCheckXKCookieResponses(t *testing.T) {
	for _, postgraduate := range []bool{false, true} {
		name, success := "undergraduate", `{"code":0,"data":[]}`
		if postgraduate {
			name, success = "postgraduate", `{"items":[],"currentPage":1}`
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name, body, location string
				status               int
				valid, wantErr       bool
			}{
				{name: "authenticated", status: 200, body: success, valid: true},
				{name: "single quoted login form", status: 200, body: `<form><input name='yhm'><input name='mm'></form>`},
				{name: "null business code", status: 200, body: `{"code":null,"data":[]}`, wantErr: true},
				{name: "CAS page 200", status: 200, body: `<form><input name="username"><input name="password"></form>`},
				{name: "JS login 200", status: 200, body: `<script>window.location.href='https://account.ccnu.edu.cn/cas/login?service=xxx'</script>`},
				{name: "graduate login 200", status: 200, body: `<form action="/yjsxt/xtgl/login_slogin.html"><input name="yhm"><input name="mm"></form>`},
				{name: "CAS redirect", status: 302, location: "https://account.ccnu.edu.cn/cas/login?service=xxx"},
				{name: "graduate redirect", status: 302, location: "https://grd.ccnu.edu.cn/yjsxt/xtgl/login_slogin.html"},
				{name: "unauthorized", status: 401},
				{name: "unknown redirect", status: 302, location: "/maintenance", wantErr: true},
				{name: "proxy authentication", status: 407, wantErr: true},
				{name: "forbidden", status: 403, wantErr: true},
				{name: "upstream error", status: 503, body: "unavailable", wantErr: true},
				{name: "unrecognized HTML", status: 200, body: "<html>maintenance</html>", wantErr: true},
				{name: "empty object", status: 200, body: `{}`, wantErr: true},
				{name: "unknown business code", status: 200, body: `{"code":9,"data":[]}`, wantErr: true},
				{name: "null result", status: 200, body: `{"code":0,"data":null,"items":null}`, wantErr: true},
				{name: "oversized", status: 200, body: strings.Repeat("x", maxXKProbeBody+1), wantErr: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					body := &cookieResponseBody{Reader: strings.NewReader(tc.body)}
					calls := 0
					client := &http.Client{Timeout: time.Second, Transport: cookieRoundTripper(func(req *http.Request) (*http.Response, error) {
						calls++
						if req.Header.Get("Cookie") != "test-cookie" {
							t.Error("missing probe cookie")
						}
						if postgraduate {
							if req.URL.Host != "grd.ccnu.edu.cn" || req.Method != http.MethodPost {
								t.Error("wrong graduate target")
							}
							if err := req.ParseForm(); err != nil || req.Form.Get("queryModel.showCount") != "1" {
								t.Error("wrong graduate probe form")
							}
						} else if req.URL.Host != "bkzhjw.ccnu.edu.cn" || req.URL.Query().Get("pageSize") != "1" {
							t.Error("wrong undergraduate target")
						}
						return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": []string{tc.location}}, Body: body}, nil
					})}
					var valid bool
					var err error
					if postgraduate {
						valid, err = NewPostGraduate(client).CheckCookie(context.Background(), "test-cookie")
					} else {
						valid, err = NewUnderGrad(client).CheckCookie(context.Background(), "test-cookie")
					}
					if valid != tc.valid || (err != nil) != tc.wantErr {
						t.Fatalf("valid=%v err=%v", valid, err)
					}
					if calls != 1 {
						t.Fatalf("probe followed redirect: calls=%d", calls)
					}
					if !body.closed {
						t.Error("response body was not closed")
					}
				})
			}
		})
	}
}

func TestCheckXKCookieCancellationAndTimeout(t *testing.T) {
	for _, postgraduate := range []bool{false, true} {
		for _, canceled := range []bool{false, true} {
			ctx, cancel := context.WithCancel(context.Background())
			if canceled {
				cancel()
			}
			client := &http.Client{Timeout: 10 * time.Millisecond, Transport: cookieRoundTripper(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			})}
			var valid bool
			var err error
			if postgraduate {
				valid, err = NewPostGraduate(client).CheckCookie(ctx, "test-cookie")
			} else {
				valid, err = NewUnderGrad(client).CheckCookie(ctx, "test-cookie")
			}
			cancel()
			want := context.DeadlineExceeded
			if canceled {
				want = context.Canceled
			}
			if valid || !errors.Is(err, want) {
				t.Fatalf("postgraduate=%v canceled=%v valid=%v err=%v", postgraduate, canceled, valid, err)
			}
		}
	}
}

func TestPostgraduateCookiePreservesPasswordError(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(postgraduateURL + "/yjsxt")
	jar.SetCookies(u, []*http.Cookie{{Name: "JSESSIONID", Value: "test-session"}, {Name: "route", Value: "test-route"}})
	client := &http.Client{Jar: jar, Transport: cookieRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("用户名或密码不正确"))}, nil
	})}
	_, err = NewPostGraduate(client).GetCookie(context.Background(), "2024100000", "test-password", &key.PublicKey)
	if !errors.Is(err, INCorrectPASSWORD) {
		t.Fatalf("err=%v, want password error", err)
	}
}

func TestEmptyCookieDoesNotProbe(t *testing.T) {
	client := &http.Client{Transport: cookieRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("empty cookie must not make an upstream request")
		return nil, nil
	})}
	for _, check := range []func(context.Context, string) (bool, error){NewUnderGrad(client).CheckCookie, NewPostGraduate(client).CheckCookie} {
		if valid, err := check(context.Background(), ""); valid || err != nil {
			t.Fatalf("valid=%v err=%v", valid, err)
		}
	}
}
