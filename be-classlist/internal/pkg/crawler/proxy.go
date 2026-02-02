package crawler

import (
	"context"
	"net/url"
	"sync"
	"time"

	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"golang.org/x/sync/singleflight"
)

type ProxyGetter interface {
	GetProxy(ctx context.Context) *url.URL
}

type proxyGetter struct {
	pc                  proxyv1.ProxyClient
	lastUpdateProxyTime int64 // 上一次更新代理时间的秒级时间戳
	updateInterval      int64 // 更新代理间隔(s)

	proxyMutex sync.RWMutex
	proxy      *url.URL

	sfGroup singleflight.Group
}

// NewProxyGetter 初始化
func NewProxyGetter(pc proxyv1.ProxyClient) ProxyGetter {
	return &proxyGetter{
		pc:                  pc,
		lastUpdateProxyTime: -1,
		updateInterval:      160,
	}
}

// GetProxy 获取代理
func (p *proxyGetter) GetProxy(ctx context.Context) *url.URL {
	currentTime := time.Now().Unix()

	// 如果缓存有效，直接返回
	p.proxyMutex.RLock()
	if currentTime-p.lastUpdateProxyTime <= p.updateInterval {
		prx := p.proxy
		p.proxyMutex.RUnlock()
		return prx
	}
	p.proxyMutex.RUnlock()

	// 代理过期，进入合并请求流程
	// 使用 DoChan 得到一个 channel，用 select 控制超时
	resultCh := p.sfGroup.DoChan("fetch_proxy", func() (interface{}, error) {
		return p.doFetchProxy(ctx)
	})

	select {
	case res := <-resultCh:
		// 如果 成功返回，返回新代理
		if res.Err == nil && res.Val != nil {
			return res.Val.(*url.URL)
		}
		// 如果 失败，降级
		return p.getOldProxy()

	case <-time.After(500 * time.Millisecond):
		// 如果 500ms 还没拿到结果，不等了，直接降级
		// 这样即使阻塞了，协程也能继续工作
		return p.getOldProxy()

	case <-ctx.Done():
		return p.getOldProxy()
	}
}

// doFetchProxy 实际执行 RPC 请求的方法
func (p *proxyGetter) doFetchProxy(ctx context.Context) (*url.URL, error) {
	// 再次 Double Check，防止重复请求
	p.proxyMutex.RLock()
	if time.Now().Unix()-p.lastUpdateProxyTime <= p.updateInterval {
		prx := p.proxy
		p.proxyMutex.RUnlock()
		return prx, nil
	}
	p.proxyMutex.RUnlock()

	logh := logger.GetLoggerFromCtx(ctx)

	rpcCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := p.pc.GetProxyAddr(rpcCtx, &proxyv1.GetProxyAddrRequest{})

	p.proxyMutex.Lock()
	defer p.proxyMutex.Unlock()

	// 无论成功还是失败，都更新时间，防止在失败的情况下高频重试 RPC
	p.lastUpdateProxyTime = time.Now().Unix()

	if err != nil || res == nil || len(res.Addr) == 0 {
		logh.Errorf("pull proxy addr failed: %v", err)
		// 失败时设置为nil，防止后续一直使用失效代理
		p.proxy = nil
		return p.proxy, err
	}

	proxyURL, parseErr := url.Parse(res.Addr)
	if parseErr != nil {
		logh.Errorf("parse proxy addr %s failed: %v", res.Addr, parseErr)
		// 失败时设置为nil，防止后续一直使用失效代理
		p.proxy = nil
		return p.proxy, parseErr
	}

	// 成功，更新代理
	p.proxy = proxyURL
	return p.proxy, nil
}

// getOldProxy 降级逻辑：获取当前持有的（可能是旧的）代理
func (p *proxyGetter) getOldProxy() *url.URL {
	p.proxyMutex.RLock()
	defer p.proxyMutex.RUnlock()
	return p.proxy
}
