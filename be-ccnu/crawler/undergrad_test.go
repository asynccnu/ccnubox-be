package crawler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// 随便写的,比较随意
func Test_GetCookie(t *testing.T) {
	p := NewPassport(NewCrawlerClient(10*time.Second, "")) // 这里测试可以不传
	ctx := context.Background()
	_, err := p.LoginPassport(ctx, "", "")
	if err != nil {
		return
	}

	ug := NewUnderGrad(p.Client)
	err = ug.LoginUnderGradSystem(ctx)
	if err != nil {
		return
	}
	cookie, err := ug.GetCookieFromUnderGradSystem()
	if err != nil {
		return
	}
	fmt.Printf("cookie:%s", cookie)
	t.Log(cookie)
}

func Test_GetLibraryCookie(t *testing.T) {
	p := NewPassport(NewCrawlerClient(10*time.Second, "")) // 这里测试可以不传
	ctx := context.Background()
	_, err := p.LoginPassport(ctx, "2024214743", "Xll810929")
	if err != nil {
		return
	}

	l := NewLibrary(p.Client)
	err = l.LoginLibrary(ctx)
	if err != nil {
		panic(err)
	}
	//token, err := l.GetSeatAuthTokenFromLibrary(ctx)
	//if err != nil {
	//	panic(err)
	//}
	//t.Log(token)
	//
	//ok, err := l.CheckLibrarySeatToken(ctx, token)
	//if err != nil {
	//	panic(err)
	//}
	//t.Log(ok)

	discussionToken, err := l.GetDiscussionAuthTokenFromLibrary(ctx)
	if err != nil {
		panic(err)
	}
	t.Log(discussionToken)

	valid, err := l.CheckLibraryDiscussionToken(ctx, discussionToken)
	if err != nil {
		panic(err)
	}
	t.Log(valid)
}
