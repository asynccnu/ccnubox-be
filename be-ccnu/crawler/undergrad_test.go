package crawler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// 随便写的,比较随意
func Test_GetCookie(t *testing.T) {
	p := NewPassport(NewCrawlerClient(10 * time.Second)) // 这里测试可以不传
	ctx := context.Background()
	_, err := p.LoginPassport(ctx, "xxx", "xxx")
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
	fmt.Printf("cookie:%s\n", cookie)
	t.Log(cookie)
}

func Test_GetLibraryCookie(t *testing.T) {
	p := NewPassport(NewCrawlerClient(10 * time.Second)) // 这里测试可以不传
	ctx := context.Background()
	_, err := p.LoginPassport(ctx, "xxx", "xxx")
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
	t.Log(discussionToken)

	valid, err := l.CheckLibraryDiscussionToken(ctx, discussionToken)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(valid)
}
