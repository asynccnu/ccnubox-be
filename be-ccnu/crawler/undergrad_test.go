package crawler

import (
	"context"
	"os"
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
