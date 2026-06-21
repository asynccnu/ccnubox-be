package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

var failoverTransportSequence atomic.Uint64

// FailoverTransport pins a client session to one proxy. If that proxy is
// rejected before the target receives the request, it retries through the
// backup proxy and pins subsequent requests to the working address.
type FailoverTransport struct {
	options []RoundTripperOption
	direct  *HttpTransport

	mu           sync.Mutex
	pinned       string
	preferBackup bool
	transports   map[string]*HttpTransport
}

func NewFailoverTransport(options ...RoundTripperOption) *FailoverTransport {
	direct := globalProxy.NewProxyTransport()
	direct.Use(options...)
	return &FailoverTransport{
		options:      options,
		direct:       direct,
		preferBackup: failoverTransportSequence.Add(1)%2 == 0,
		transports:   make(map[string]*HttpTransport),
	}
}

func (t *FailoverTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	candidates := t.proxyCandidates(req)
	if len(candidates) == 0 {
		return t.direct.RoundTrip(req)
	}

	var lastResp *http.Response
	var lastErr error
	for attempt, proxyAddr := range candidates {
		attemptReq, err := cloneRequestForAttempt(req, attempt)
		if err != nil {
			if lastResp != nil || lastErr != nil {
				return lastResp, lastErr
			}
			return nil, err
		}

		transport, err := t.transportFor(proxyAddr)
		if err != nil {
			lastErr = err
			continue
		}

		resp, roundTripErr := transport.RoundTrip(attemptReq)
		lastResp, lastErr = resp, roundTripErr
		if !isProxyFailure(resp, roundTripErr) {
			t.pin(proxyAddr)
			return resp, roundTripErr
		}

		t.unpin(proxyAddr)
		if resp != nil && resp.Body != nil && attempt+1 < len(candidates) {
			_ = resp.Body.Close()
		}
	}

	return lastResp, lastErr
}

func (t *FailoverTransport) proxyCandidates(req *http.Request) []string {
	t.mu.Lock()
	pinned := t.pinned
	t.mu.Unlock()

	addrs := globalProxy.GetProxyAddr(req.Context(), 2)
	if pinned == "" && t.preferBackup && len(addrs) > 1 {
		addrs[0], addrs[1] = addrs[1], addrs[0]
	}
	result := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	for _, addr := range append([]string{pinned}, addrs...) {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if _, exists := seen[addr]; exists {
			continue
		}
		seen[addr] = struct{}{}
		result = append(result, addr)
	}
	return result
}

func (t *FailoverTransport) transportFor(proxyAddr string) (*HttpTransport, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if transport, ok := t.transports[proxyAddr]; ok {
		return transport, nil
	}

	proxyURL, err := url.Parse(proxyAddr)
	if err != nil || proxyURL.Scheme == "" || proxyURL.Host == "" {
		return nil, fmt.Errorf("invalid proxy URL")
	}

	transport := globalProxy.NewProxyTransport()
	transport.Proxy = http.ProxyURL(proxyURL)
	transport.Use(t.options...)
	t.transports[proxyAddr] = transport
	return transport, nil
}

func (t *FailoverTransport) pin(proxyAddr string) {
	t.mu.Lock()
	t.pinned = proxyAddr
	t.mu.Unlock()
}

func (t *FailoverTransport) unpin(proxyAddr string) {
	t.mu.Lock()
	if t.pinned == proxyAddr {
		t.pinned = ""
	}
	t.mu.Unlock()
}

func cloneRequestForAttempt(req *http.Request, attempt int) (*http.Request, error) {
	if attempt == 0 {
		return req, nil
	}

	clone := req.Clone(req.Context())
	if req.Body == nil {
		return clone, nil
	}
	if req.GetBody == nil {
		return nil, errors.New("proxy failover: request body cannot be replayed")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("proxy failover: recreate request body: %w", err)
	}
	clone.Body = body
	return clone, nil
}

func isProxyFailure(resp *http.Response, err error) bool {
	if resp != nil && resp.StatusCode == http.StatusProxyAuthRequired {
		return true
	}
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "proxy authentication failed") ||
		strings.Contains(message, "proxy authentication required") ||
		strings.Contains(message, "proxyconnect tcp")
}
