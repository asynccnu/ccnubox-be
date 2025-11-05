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
	fmt.Println(cookie)
	t.Log(cookie)
}
