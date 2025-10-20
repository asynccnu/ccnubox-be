package crawler

import (
	"context"
	"testing"
	"time"
)

// 随便写的,比较随意
func Test_GetCookie(t *testing.T) {
	ug := NewUnderGrad(NewCrawlerClient(10 * time.Second))
	ctx := context.Background()
	html, err := ug.GetParamsFromHtml(ctx)
	if err != nil {
		return
	}
	err = ug.LoginCCNUPassport(ctx, "2023215153", "Sjn3265926531", html)
	if err != nil {
		return
	}
	err = ug.LoginUnderGradSystem(ctx)
	if err != nil {
		return
	}
	cookie, err := ug.GetCookieFromUnderGradSystem()
	if err != nil {
		return
	}
	t.Log(cookie)
}
