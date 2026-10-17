package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"go.uber.org/zap"
)

func Test_ccnuService_getGradCookie(t *testing.T) {
	testLogger := zapx.NewZapLogger(zap.NewNop())
	ccs := NewCCNUService(testLogger, proxy.NewDirectHttpProxy(nil), "test-secret")
	stuId, password := "", ""
	_, err := ccs.GetLibraryToken(context.Background(), stuId, password, ccnuv1.LIBRARY_TYPE_LIBRARY_SEAT)
	if err == nil {
		t.Fatal("expected invalid student id error")
	}
}

func TestXKSystemContract(t *testing.T) {
	for _, tc := range []struct {
		studentID, raw string
		want           tool.XKSystem
		wantErr        bool
	}{
		{"2024200000", "", tool.XKUndergraduate, false},
		{"2024100000", "", tool.XKPostgraduate, false},
		{"2024000000", "", tool.XKPostgraduate, false},
		{"2024200000", "new", tool.XKUndergraduate, false},
		{"2024200000", "undergraduate", tool.XKUndergraduate, false},
		{"2024100000", "postgraduate", tool.XKPostgraduate, false},
		{"2024100000", "undergraduate", "", true},
		{"2024200000", "postgraduate", "", true},
		{"2024100000", "new", "", true},
		{"2024200000", "old", "", true},
		{"2024200000", "unknown", "", true},
		{"invalid", "", "", true},
	} {
		system, err := tool.ResolveXKSystem(tc.studentID, tc.raw)
		if tc.wantErr {
			if !ccnuv1.IsInvalidXkSystem(err) {
				t.Fatalf("sid=%s type=%s err=%v", tc.studentID, tc.raw, err)
			}
		} else if err != nil || system != tc.want {
			t.Fatalf("sid=%s type=%s system=%s err=%v", tc.studentID, tc.raw, system, err)
		}
	}
	service := &ccnuService{}
	for _, system := range []tool.XKSystem{"", "new", "unknown"} {
		if _, err := service.CheckXKCookie(context.Background(), "cookie", system); !ccnuv1.IsInvalidXkSystem(err) {
			t.Fatalf("err=%v", err)
		}
	}
	if _, err := service.GetXKCookie(context.Background(), "2024200000", "password", tool.XKPostgraduate); !ccnuv1.IsInvalidXkSystem(err) {
		t.Fatalf("err=%v", err)
	}
}

type xkRoundTripper func(*http.Request) (*http.Response, error)

func (f xkRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type xkProxy struct{ transport http.RoundTripper }

func (p xkProxy) NewProxyClient(options ...proxy.Option) *proxy.HttpClient {
	client := &proxy.HttpClient{Client: &http.Client{}}
	client.Use(options...)
	client.Transport = p.transport
	return client
}

func TestCheckXKCookieDispatchAndFailure(t *testing.T) {
	for _, system := range []tool.XKSystem{tool.XKUndergraduate, tool.XKPostgraduate} {
		for _, code := range []int{200, 401, 503} {
			service := &ccnuService{timeout: time.Second, p: xkProxy{transport: xkRoundTripper(func(req *http.Request) (*http.Response, error) {
				body, host := `{"code":0,"data":[]}`, "bkzhjw.ccnu.edu.cn"
				if system == tool.XKPostgraduate {
					body, host = `{"items":[]}`, "grd.ccnu.edu.cn"
				}
				if req.URL.Host != host {
					t.Fatalf("system=%s host=%s", system, req.URL.Host)
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			valid, err := service.CheckXKCookie(context.Background(), "cookie", system)
			if valid != (code == 200) || (err != nil) != (code == 503) {
				t.Fatalf("system=%s status=%d valid=%v err=%v", system, code, valid, err)
			}
			if code == 503 && !ccnuv1.IsCcnuserverError(err) {
				t.Fatalf("err=%v, want dependency error", err)
			}
		}
	}
}

func TestGetGraduateCookieClassifiesPasswordError(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	loginCalls := 0
	service := &ccnuService{timeout: time.Second, p: xkProxy{transport: xkRoundTripper(func(req *http.Request) (*http.Response, error) {
		var body string
		if strings.HasSuffix(req.URL.Path, "login_getPublicKey.html") {
			body = fmt.Sprintf(`{"modulus":%q,"exponent":"AQAB"}`, base64.StdEncoding.EncodeToString(key.N.Bytes()))
		} else {
			loginCalls++
			body = "用户名或密码不正确"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	_, err = service.GetXKCookie(context.Background(), "2024100000", "test-password", tool.XKPostgraduate)
	if !ccnuv1.IsInvalidSidOrPwd(err) || loginCalls != 1 {
		t.Fatalf("err=%v loginCalls=%d", err, loginCalls)
	}
}
