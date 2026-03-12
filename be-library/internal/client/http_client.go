package client

import (
	"context"
	"net/http"
	"time"
)

type Client interface {
	DoWithContext(context.Context, *http.Request) (*http.Response, error)
}

type HttpClient struct {
	Client *http.Client
}

func NewHttpClient() Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   false,
		},
		Timeout: 30 * time.Second,
	}

	return &HttpClient{
		Client: client,
	}

}

func (hcli *HttpClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	return hcli.Client.Do(req)
}
