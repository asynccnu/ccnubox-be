package crawler

import (
	"net/http"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
)

func TestNewCrawlerClientWithCookieJarUsesProxyTransport(t *testing.T) {
	client := NewCrawlerClientWithCookieJar(time.Second, nil, &proxy.HttpProxy{})

	if client.Transport == nil {
		t.Fatal("crawler client transport is nil; requests would bypass be-proxy")
	}
	if _, ok := client.Transport.(*proxy.HttpTransport); !ok {
		t.Fatalf("crawler client transport = %T, want *proxy.HttpTransport", client.Transport)
	}
	if client.CheckRedirect == nil {
		t.Fatal("crawler client redirect policy is nil")
	}
	if err := client.CheckRedirect(&http.Request{}, nil); err != nil {
		t.Fatalf("crawler client should allow redirects, got %v", err)
	}
}
