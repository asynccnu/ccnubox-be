package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

type ShenLongProxy struct {
	Api          string
	Addr         string
	PollInterval int
	RetryCount   int

	mu sync.RWMutex // 异步写+并发读
}

var (
	once          sync.Once // 保证只初始化一次
	shenLongProxy *ShenLongProxy
)

func InitShenLongProxy() {
	var config struct {
		Api      string `json:"api"`
		Interval int    `json:"interval"`
		Retry    int    `json:"retry"`
	}
	if err := viper.UnmarshalKey("shenlong", &config); err != nil {
		panic(err)
	}

	if config.Api == "" {
		log.Warnf("use DefaultClient due to the empty of proxy setting (time:%s)", time.Now())
		return
	}

	shenLongProxy = &ShenLongProxy{
		Api:          config.Api,
		PollInterval: config.Interval,
		RetryCount:   config.Retry,
	}
	// 初始化之后就要马上更新一次ip, 保证不是空的
	shenLongProxy.fetchIp()

	c := cron.New()
	c.AddFunc(fmt.Sprintf("@every %ds", shenLongProxy.PollInterval), shenLongProxy.fetchIp)
	c.Start()
}

func NewShenLongHTTPClient() *http.Client {
	// 懒初始化, 使用时才初始化, 保证尽可能只在调用方切换函数
	if shenLongProxy == nil {
		once.Do(InitShenLongProxy)
	}

	// 未配置代理时使用默认client
	if shenLongProxy == nil || shenLongProxy.Api == "" {
		log.Warnf("use DefaultClient due to the empty of proxy setting")
		return NewHTTPClient()
	}

	// 获取代理addr
	shenLongProxy.mu.RLock()
	proxyAddr := shenLongProxy.Addr
	shenLongProxy.mu.RUnlock()

	proxy, err := url.Parse(proxyAddr)
	if err != nil {
		return NewHTTPClient()
	}

	netTransport := &http.Transport{
		Proxy:                 http.ProxyURL(proxy),
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}

	httpClient := &http.Client{
		Transport: netTransport,
	}

	return httpClient
}

func (s *ShenLongProxy) fetchIp() {
	for i := 0; i < s.RetryCount; i++ {

		resp, err := http.Get(s.Api)
		if err != nil {
			log.Errorf("fetch ip fail(attempt %d/%d): %v", i+1, s.RetryCount, err)
			// TODO: log
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("read resp when fetching ip fail(attempt %d): %v", i+1, s.RetryCount)
			continue
		}
		resp.Body.Close() // 读取完就关闭, for里面defer有资源泄漏问题

		// 如果不能正常获取ip会是{code: xx, msg: xx}的json
		if !strings.Contains(string(body), "code") {

			log.Debug("fetch ip success, now: ", time.Now())
			s.mu.Lock()
			s.Addr = wrapRes(string(body))
			s.mu.Unlock()

			break
		}

		time.Sleep(time.Second * 2)
	}

}

func wrapRes(res string) string {
	// 会返回\t\n, 提供方那边去不了
	return fmt.Sprintf("http://%s", strings.TrimSpace(res))
}

func NewHTTPClient() *http.Client {
	client := &http.Client{Transport: nil,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}
	return client
}
