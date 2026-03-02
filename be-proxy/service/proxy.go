package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-proxy/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
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

	mu sync.RWMutex // 保护 Addr 和 AddrBackup 的并发读写
	l  logger.Logger
}

var (
	ErrEmptyConfig = errorx.New("proxy: empty configuration")
)

func (s *ShenLongProxy) GetProxyAddr(_ context.Context) (string, string, error) {
	// 未配置代理时返回错误
	if s.Api == "" {
		return "", "", ErrEmptyConfig
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// 如果当前地址为空，说明 fetchIp 尚未成功执行过
	if s.Addr == "" {
		return "", "", errorx.New("proxy: no available proxy address currently")
	}

	return s.Addr, s.AddrBackup, nil
}

func NewProxyService(l logger.Logger, cfg *conf.ServerConf) ProxyService {
	if cfg.ShenLongConf.API == "" {
		l.Warn("proxy: fail to new client because API is empty", logger.String("time", time.Now().Format(time.DateTime)))
		panic(ErrEmptyConfig)
	}

	s := &ShenLongProxy{
		Api:          cfg.ShenLongConf.API,
		PollInterval: cfg.ShenLongConf.Interval,
		RetryCount:   cfg.ShenLongConf.Retry,
		Username:     cfg.ShenLongConf.Username,
		Password:     cfg.ShenLongConf.Password,
		l:            l,
	}

	// 初始化后同步执行一次更新，确保启动时就有 IP 可用
	s.fetchIp()

	// 注册定时更新任务
	c := cron.New(cron.WithSeconds())
	spec := fmt.Sprintf("@every %ds", s.PollInterval)
	_, err := c.AddFunc(spec, s.fetchIp)
	if err != nil {
		l.Error("proxy: failed to add cron task", logger.Error(err))
		// 如果定时任务注册失败，应根据业务严重性决定是否 panic
	}
	c.Start()

	return s
}

func (s *ShenLongProxy) fetchIp() {
	client := &http.Client{Timeout: 10 * time.Second}

	var lastErr error

	for i := 0; i < s.RetryCount; i++ {
		resp, err := client.Get(s.Api)
		if err != nil {
			lastErr = err
			s.l.Error("proxy: fetch ip request failed",
				logger.Error(err),
				logger.Int("attempt", i+1),
			)
			time.Sleep(2 * time.Second)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close() // 及时关闭资源，防止 for 循环内泄露

		if err != nil {
			lastErr = err
			s.l.Error("proxy: read response body failed",
				logger.Error(err),
				logger.Int("attempt", i+1),
			)
			time.Sleep(2 * time.Second)
			continue
		}

		bodyStr := string(body)
		// 校验返回内容是否为正常的 IP 列表（通常代理商报错会返回包含 "code" 的 JSON）
		if !strings.Contains(bodyStr, "code") && strings.Contains(bodyStr, ".") {
			// 处理 Windows 风格的换行符
			bodyStr = strings.ReplaceAll(bodyStr, "\r\n", "\n")
			addrs := strings.Split(strings.TrimSpace(bodyStr), "\n")

			if len(addrs) > 0 {
				s.mu.Lock()
				s.Addr = s.wrapRes(addrs[0])
				// 容错处理：如果只返回了一个 IP，则备份 IP 与主 IP 一致
				if len(addrs) > 1 {
					s.AddrBackup = s.wrapRes(addrs[1])
				} else {
					s.AddrBackup = s.Addr
				}
				s.mu.Unlock()

				s.l.Info("proxy: fetch ip success", logger.String("primary", addrs[0]))
				return // 成功后直接退出重试循环
			}
		}

		lastErr = fmt.Errorf("invalid response: %s", bodyStr)
		s.l.Error("proxy: invalid response from api",
			logger.String("resp", bodyStr),
			logger.Int("attempt", i+1),
		)
		time.Sleep(2 * time.Second)
	}

	s.mu.Lock()
	s.Addr = ""
	s.AddrBackup = ""
	s.mu.Unlock()

	if lastErr != nil {
		s.l.Error("proxy: all attempts failed to fetch new ip",
			logger.Int("max_retries", s.RetryCount),
			logger.Error(lastErr),
		)
	} else {
		s.l.Error("proxy: all attempts failed to fetch new ip", logger.Int("max_retries", s.RetryCount))
	}
}

func (s *ShenLongProxy) wrapRes(res string) string {
	// 代理商返回的 IP 经常带有不可见字符或空白符，需要彻底清理
	cleanRes := strings.TrimSpace(res)
	if cleanRes == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%s@%s", s.Username, s.Password, cleanRes)
}
