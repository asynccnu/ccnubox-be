package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ShenLongProxy struct {
	Api          string
	Addr         string
	PollInterval int
	RetryCount   int
	Username     string
	Password     string

	mu sync.RWMutex // 异步写+并发读
}

var (
	once sync.Once // 保证只初始化一次

	ErrEmptyConfig = errors.New("empty config")
)

func (s *ShenLongProxy) GetProxyAddr(_ context.Context) (string, error) {
	// 懒初始化
	if s == nil {
		once.Do(func() {
			NewProxyService()
		})
	}

	// 未配置代理时使用
	if s.Api == "" {
		log.Warnf("empty proxy setting")
		return "", ErrEmptyConfig
	}

	// 获取代理addr
	s.mu.RLock()
	proxyAddr := s.Addr
	s.mu.RUnlock()

	return proxyAddr, nil
}

func NewProxyService() ProxyService {
	var config struct {
		Api      string `json:"api"`
		Interval int    `json:"interval"`
		Retry    int    `json:"retry"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := viper.UnmarshalKey("shenlong", &config); err != nil {
		panic(err)
	}

	if config.Api == "" {
		log.Warnf("use DefualtClient due to the empty of proxy setting (time:%s)", time.Now())
		panic(ErrEmptyConfig)
	}

	s := &ShenLongProxy{
		Api:          config.Api,
		PollInterval: config.Interval,
		RetryCount:   config.Retry,
		Username:     config.Username,
		Password:     config.Password,
	}
	// 初始化之后就要马上更新一次ip, 保证不是空的
	s.fetchIp()

	c := cron.New()
	_, _ = c.AddFunc(fmt.Sprintf("@every %ds", s.PollInterval), s.fetchIp)
	c.Start()

	return s
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
		_ = resp.Body.Close() // 读取完就关闭, for里面defer有资源泄漏问题

		// 如果不能正常获取ip会是{code: xx, msg: xx}的json
		if !strings.Contains(string(body), "code") {

			log.Debug("fetch ip success, now: ", time.Now())
			s.mu.Lock()
			s.Addr = s.wrapRes(string(body))
			s.mu.Unlock()

			break
		}

		time.Sleep(time.Second * 2)
	}

}

func (s *ShenLongProxy) wrapRes(res string) string {
	// 会返回\t\n, 提供方那边去不了
	return fmt.Sprintf("http://%s:%s@%s", s.Username, s.Password, strings.TrimSpace(res))
}
