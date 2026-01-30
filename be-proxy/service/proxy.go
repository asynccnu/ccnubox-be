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

	"github.com/asynccnu/ccnubox-be/be-proxy/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
)

type ShenLongProxy struct {
	Api          string
	Addr         string
	AddrBackup   string
	Username     string
	Password     string
	PollInterval int
	RetryCount   int

	mu sync.RWMutex // 异步写+并发读
	l  logger.Logger
}

var (
	ErrEmptyConfig = errors.New("empty config")
)

func (s *ShenLongProxy) GetProxyAddr(_ context.Context) (string, string, error) {
	// 未配置代理时使用
	if s.Api == "" {
		log.Warnf("empty proxy setting")
		return "", "", ErrEmptyConfig
	}

	// 获取代理addr
	s.mu.RLock()
	proxyAddr := s.Addr
	proxyAddrBackup := s.AddrBackup
	s.mu.RUnlock()

	return proxyAddr, proxyAddrBackup, nil
}

func NewProxyService(l logger.Logger, cfg *conf.ServerConf) ProxyService {
	if cfg.ShenLongConf.API == "" {
		log.Warnf("use DefualtClient due to the empty of proxy setting (time:%s)", time.Now())
		panic(ErrEmptyConfig)
	}

	s := &ShenLongProxy{
		Api:          cfg.ShenLongConf.API,
		PollInterval: cfg.ShenLongConf.Interval,
		RetryCount:   cfg.ShenLongConf.Retry,
		Username:     cfg.ShenLongConf.Username,
		Password:     cfg.ShenLongConf.Password,

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
			addrs := strings.Split(string(body), "\r\n")

			s.mu.Lock()
			s.Addr = s.wrapRes(addrs[0])
			s.AddrBackup = s.wrapRes(addrs[1])
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
