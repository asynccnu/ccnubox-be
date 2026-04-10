//go:build integration

package crawler

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func requireEnv(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Skipf("skip integration test: %s is not set", key)
	}
	return value
}

func requirePassportCredentials(t *testing.T) (string, string) {
	t.Helper()
	return requireEnv(t, "CCNU_TEST_STU_ID"), requireEnv(t, "CCNU_TEST_PASSWORD")
}

func requirePostgraduateCredentials(t *testing.T) (string, string) {
	t.Helper()
	return requireEnv(t, "CCNU_TEST_POSTGRAD_ID"), requireEnv(t, "CCNU_TEST_POSTGRAD_PASSWORD")
}

func newIntegrationContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 40*time.Second)
}

func TestIntegrationPassportGetParamsFromHtml(t *testing.T) {
	_, _ = requirePassportCredentials(t)

	ctx, cancel := newIntegrationContext()
	defer cancel()

	passport := NewPassport(NewCrawlerClient(20 * time.Second))
	t.Logf("passport.Client.Transport.(*http.Transport): %+v", passport.Client.Transport.(*http.Transport))
	params, err := passport.getParamsFromHtml(ctx)
	if err != nil {
		t.Fatalf("getParamsFromHtml() error = %v", err)
	}

	if params.lt == "" || params.execution == "" || params._eventId == "" || params.JSESSIONID == "" {
		t.Fatalf("unexpected empty params: %+v", params)
	}
	t.Logf("extracted params: %+v", params)
}

func TestIntegrationPassportExtractField(t *testing.T) {
	_, _ = requirePassportCredentials(t)

	ctx, cancel := newIntegrationContext()
	defer cancel()

	client := NewCrawlerClient(20 * time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LoginCCNUPassPortURL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	bodyStr := string(body)
	lt, err := extractField(bodyStr, `name="lt".+value="(.+)"`, "lt")
	if err != nil {
		t.Fatalf("extractField(lt) error = %v", err)
	}
	execution, err := extractField(bodyStr, `name="execution".+value="(.+)"`, "execution")
	if err != nil {
		t.Fatalf("extractField(execution) error = %v", err)
	}
	eventID, err := extractField(bodyStr, `name="_eventId".+value="(.+)"`, "_eventId")
	if err != nil {
		t.Fatalf("extractField(_eventId) error = %v", err)
	}

	if lt == "" || execution == "" || eventID == "" {
		t.Fatalf("unexpected extracted values: lt=%q execution=%q eventID=%q", lt, execution, eventID)
	}
}

func TestIntegrationPassportLoginCCNUPassport(t *testing.T) {
	stuID, password := requirePassportCredentials(t)

	ctx, cancel := newIntegrationContext()
	defer cancel()

	client := NewCrawlerClient(20 * time.Second)
	passport := NewPassport(client)

	params, err := passport.getParamsFromHtml(ctx)
	if err != nil {
		t.Fatalf("getParamsFromHtml() error = %v", err)
	}

	if err := passport.loginCCNUPassport(ctx, stuID, password, params); err != nil {
		t.Fatalf("loginCCNUPassport() error = %v", err)
	}
}

func TestIntegrationPassportLoginPassport(t *testing.T) {
	stuID, password := requirePassportCredentials(t)

	ctx, cancel := newIntegrationContext()
	defer cancel()

	passport := NewPassport(NewCrawlerClient(20 * time.Second))
	ok, err := passport.LoginPassport(ctx, stuID, password)
	if err != nil {
		t.Fatalf("LoginPassport() error = %v", err)
	}
	if !ok {
		t.Fatal("LoginPassport() returned false")
	}
}

func TestIntegrationUnderGradLoginUnderGradSystem(t *testing.T) {
	stuID, password := requirePassportCredentials(t)

	ctx, cancel := newIntegrationContext()
	defer cancel()

	client := NewCrawlerClient(20 * time.Second)
	passport := NewPassport(client)

	ok, err := passport.LoginPassport(ctx, stuID, password)
	if err != nil {
		t.Fatalf("LoginPassport() error = %v", err)
	}
	if !ok {
		t.Fatal("LoginPassport() returned false")
	}

	ug := NewUnderGrad(client)
	if err := ug.LoginUnderGradSystem(ctx); err != nil {
		t.Fatalf("LoginUnderGradSystem() error = %v", err)
	}
}

func TestIntegrationUnderGradGetCookieFromUnderGradSystem(t *testing.T) {
	stuID, password := requirePassportCredentials(t)

	ctx, cancel := newIntegrationContext()
	defer cancel()

	client := NewCrawlerClient(20 * time.Second)
	passport := NewPassport(client)

	ok, err := passport.LoginPassport(ctx, stuID, password)
	if err != nil {
		t.Fatalf("LoginPassport() error = %v", err)
	}
	if !ok {
		t.Fatal("LoginPassport() returned false")
	}

	ug := NewUnderGrad(client)
	if err := ug.LoginUnderGradSystem(ctx); err != nil {
		t.Fatalf("LoginUnderGradSystem() error = %v", err)
	}

	cookie, err := ug.GetCookieFromUnderGradSystem()
	if err != nil {
		t.Fatalf("GetCookieFromUnderGradSystem() error = %v", err)
	}
	if strings.TrimSpace(cookie) == "" {
		t.Fatal("empty undergrad cookie")
	}
}

func TestIntegrationLibrary(t *testing.T) {
	stuID, password := requirePassportCredentials(t)
	secret := requireEnv(t, "CCNU_TEST_LIBRARY_SECRET")

	ctx, cancel := newIntegrationContext()
	defer cancel()

	client := NewCrawlerClient(20 * time.Second)
	passport := NewPassport(client)

	ok, err := passport.LoginPassport(ctx, stuID, password)
	if err != nil {
		t.Fatalf("LoginPassport() error = %v", err)
	}
	if !ok {
		t.Fatal("LoginPassport() returned false")
	}

	library := NewLibrary(client, secret)
	if err := library.LoginLibrary(ctx); err != nil {
		t.Fatalf("LoginLibrary() error = %v", err)
	}

	seatToken, err := library.GetSeatAuthTokenFromLibrary(ctx)
	if err != nil {
		t.Fatalf("GetSeatAuthTokenFromLibrary() error = %v", err)
	}
	if seatToken == "" {
		t.Fatal("empty seat token")
	}

	seatOK, err := library.CheckLibrarySeatToken(ctx, seatToken)
	if err != nil {
		t.Fatalf("CheckLibrarySeatToken() error = %v", err)
	}
	if !seatOK {
		t.Fatal("seat token is invalid")
	}

	discussionToken, err := library.GetDiscussionAuthTokenFromLibrary(ctx)
	if err != nil {
		t.Fatalf("GetDiscussionAuthTokenFromLibrary() error = %v", err)
	}
	if discussionToken == "" {
		t.Fatal("empty discussion token")
	}

	discussionOK, err := library.CheckLibraryDiscussionToken(ctx, discussionToken)
	if err != nil {
		t.Fatalf("CheckLibraryDiscussionToken() error = %v", err)
	}
	if !discussionOK {
		t.Fatal("discussion token is invalid")
	}
}

func TestIntegrationPostGraduate(t *testing.T) {
	stuID, password := requirePostgraduateCredentials(t)

	ctx, cancel := newIntegrationContext()
	defer cancel()

	client := NewCrawlerClient(20 * time.Second)
	pg := NewPostGraduate(client)

	pubKey, err := pg.FetchPublicKey(ctx)
	if err != nil {
		t.Fatalf("FetchPublicKey() error = %v", err)
	}

	if err := pg.LoginPostgraduateSystem(ctx, stuID, password, pubKey); err != nil {
		t.Fatalf("LoginPostgraduateSystem() error = %v", err)
	}

	cookie, err := pg.GetCookie(ctx, stuID, password, pubKey)
	if err != nil {
		t.Fatalf("GetCookie() error = %v", err)
	}
	if cookie == "" {
		t.Fatal("empty postgraduate cookie")
	}
}
