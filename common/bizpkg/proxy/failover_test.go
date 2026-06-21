package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFailoverTransportRetriesBackupAndPinsIt(t *testing.T) {
	failoverTransportSequence.Store(0)
	var primaryCalls atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		primaryCalls.Add(1)
		w.WriteHeader(http.StatusProxyAuthRequired)
	}))
	defer primary.Close()

	var backupCalls atomic.Int32
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupCalls.Add(1)
		_, _ = io.WriteString(w, "ok:"+r.Method)
	}))
	defer backup.Close()

	original := globalProxy
	globalProxy = &HttpProxy{Addr: primary.URL, AddrBackup: backup.URL}
	defer func() { globalProxy = original }()

	client := &http.Client{Transport: NewFailoverTransport()}
	for i := 0; i < 2; i++ {
		resp, err := client.Get("http://target.invalid/resource")
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i, resp.StatusCode)
		}
	}

	if got := primaryCalls.Load(); got != 1 {
		t.Fatalf("primary calls = %d, want 1", got)
	}
	if got := backupCalls.Load(); got != 2 {
		t.Fatalf("backup calls = %d, want 2", got)
	}
}

func TestFailoverTransportReplaysPostBody(t *testing.T) {
	failoverTransportSequence.Store(0)
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusProxyAuthRequired)
	}))
	defer primary.Close()

	var body string
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.WriteHeader(http.StatusOK)
	}))
	defer backup.Close()

	original := globalProxy
	globalProxy = &HttpProxy{Addr: primary.URL, AddrBackup: backup.URL}
	defer func() { globalProxy = original }()

	client := &http.Client{Transport: NewFailoverTransport()}
	req, err := http.NewRequest(http.MethodPost, "http://target.invalid/login", strings.NewReader("a=1"))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if body != "a=1" {
		t.Fatalf("backup body = %q, want %q", body, "a=1")
	}
}

func TestFailoverTransportAlternatesInitialProxy(t *testing.T) {
	original := globalProxy
	globalProxy = &HttpProxy{Addr: "http://primary:80", AddrBackup: "http://backup:80"}
	defer func() { globalProxy = original }()

	failoverTransportSequence.Store(0)
	request, err := http.NewRequest(http.MethodGet, "http://target.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	first := NewFailoverTransport().proxyCandidates(request)
	second := NewFailoverTransport().proxyCandidates(request)
	if first[0] != globalProxy.Addr {
		t.Fatalf("first client starts with %q, want primary", first[0])
	}
	if second[0] != globalProxy.AddrBackup {
		t.Fatalf("second client starts with %q, want backup", second[0])
	}
}
