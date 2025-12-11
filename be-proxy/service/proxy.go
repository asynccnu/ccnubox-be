package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

type ShenLongProxy struct {
	Api          string
	Addr         string
	PollInterval int
	RetryCount   int
	Username     string
	Password     string

	mu sync.RWMutex // 异步写+并发读
	l  logger.Logger
}

var (
	ErrEmptyConfig = errors.New("empty config")
)

func (s *ShenLongProxy) GetProxyAddr(_ context.Context) (string, error) {
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

func NewProxyService(l logger.Logger) ProxyService {
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

		l: l,
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
			s.l.Error("fetch ip fail",
				logger.Error(err),
				logger.Int("attempt", i+1),
			)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.l.Error("read resp when fetching ip fail",
				logger.Error(err),
				logger.Int("attempt", i+1),
			)
			continue
		}
		_ = resp.Body.Close() // 读取完就关闭, for里面defer有资源泄漏问题

		// 如果不能正常获取ip会是{code: xx, msg: xx}的json
		if !strings.Contains(string(body), "code") {
			s.l.Info("fetch ip success",
				logger.String("time", time.Now().Format(time.RFC3339)),
			)
			s.mu.Lock()
			s.Addr = s.wrapRes(string(body))
			s.mu.Unlock()

			break
		} else {
			s.l.Error("fetch ip fail, invalid resp",
				logger.String("resp", string(body)),
				logger.Int("attempt", i+1),
			)
		}

		time.Sleep(time.Second * 2)
	}

}

func (s *ShenLongProxy) wrapRes(res string) string {
	// 会返回\t\n, 提供方那边去不了
	return fmt.Sprintf("http://%s:%s@%s", s.Username, s.Password, strings.TrimSpace(res))
}
