package service

import (
	"context"
	"testing"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type nopLogger struct{}

func (nopLogger) WithContext(context.Context) logger.Logger { return nopLogger{} }
func (nopLogger) With(...logger.Field) logger.Logger        { return nopLogger{} }
func (nopLogger) Debug(string, ...logger.Field)             {}
func (nopLogger) Info(string, ...logger.Field)              {}
func (nopLogger) Warn(string, ...logger.Field)              {}
func (nopLogger) Error(string, ...logger.Field)             {}
func (nopLogger) Debugf(string, ...interface{})             {}
func (nopLogger) Infof(string, ...interface{})              {}
func (nopLogger) Warnf(string, ...interface{})              {}
func (nopLogger) Errorf(string, ...interface{})             {}
func (nopLogger) AddCallerSkip(int) logger.Logger           { return nopLogger{} }

func TestHttpProxyWrapResEscapesCredentials(t *testing.T) {
	svc := &HttpProxy{
		Username: "user@example.com",
		Password: "p@ss:/?#word",
		l:        nopLogger{},
	}

	got := svc.wrapRes(" 127.0.0.1:8080 \r\n")
	want := "http://user%40example.com:p%40ss%3A%2F%3F%23word@127.0.0.1:8080"
	if got != want {
		t.Fatalf("wrapRes() = %q, want %q", got, want)
	}
}

func TestHttpProxyWrapResAllowsPlainAddressWithoutAuth(t *testing.T) {
	svc := &HttpProxy{l: nopLogger{}}

	got := svc.wrapRes("127.0.0.1:8080")
	want := "http://127.0.0.1:8080"
	if got != want {
		t.Fatalf("wrapRes() = %q, want %q", got, want)
	}
}

func TestRotateAddrFirstFetch(t *testing.T) {
	svc := &HttpProxy{l: nopLogger{}}
	// Both empty initially
	if svc.Addr != "" || svc.AddrBackup != "" {
		t.Fatal("expected both empty initially")
	}

	svc.rotateAddr("http://1.2.3.4:8080")
	if svc.Addr != "http://1.2.3.4:8080" {
		t.Fatalf("Addr = %q, want http://1.2.3.4:8080", svc.Addr)
	}
	if svc.AddrBackup != "http://1.2.3.4:8080" {
		t.Fatalf("AddrBackup = %q, want http://1.2.3.4:8080 (mirrors primary on first fetch)", svc.AddrBackup)
	}
}

func TestRotateAddrNormalRotation(t *testing.T) {
	svc := &HttpProxy{
		Addr:       "http://1.2.3.4:8080",
		AddrBackup: "http://5.6.7.8:8080",
		l:          nopLogger{},
	}

	svc.rotateAddr("http://9.10.11.12:8080")
	if svc.Addr != "http://9.10.11.12:8080" {
		t.Fatalf("Addr = %q, want http://9.10.11.12:8080", svc.Addr)
	}
	if svc.AddrBackup != "http://1.2.3.4:8080" {
		t.Fatalf("AddrBackup = %q, want previous Addr http://1.2.3.4:8080", svc.AddrBackup)
	}
}

func TestRotateAddrMultipleRotations(t *testing.T) {
	svc := &HttpProxy{l: nopLogger{}}

	// first
	svc.rotateAddr("http://a:1")
	if svc.Addr != "http://a:1" || svc.AddrBackup != "http://a:1" {
		t.Fatal("first rotation mismatch")
	}

	// second
	svc.rotateAddr("http://b:2")
	if svc.Addr != "http://b:2" || svc.AddrBackup != "http://a:1" {
		t.Fatal("second rotation mismatch")
	}

	// third
	svc.rotateAddr("http://c:3")
	if svc.Addr != "http://c:3" || svc.AddrBackup != "http://b:2" {
		t.Fatal("third rotation mismatch")
	}

	// fourth
	svc.rotateAddr("http://d:4")
	if svc.Addr != "http://d:4" || svc.AddrBackup != "http://c:3" {
		t.Fatal("fourth rotation mismatch")
	}
}

func TestRotateAddrsUsesTwoFreshAddresses(t *testing.T) {
	svc := &HttpProxy{
		Addr:       "http://old-primary:1",
		AddrBackup: "http://old-backup:2",
		l:          nopLogger{},
	}
	svc.rotateAddrs("http://new-primary:3", "http://new-backup:4")
	if svc.Addr != "http://new-primary:3" {
		t.Fatalf("Addr = %q", svc.Addr)
	}
	if svc.AddrBackup != "http://new-backup:4" {
		t.Fatalf("AddrBackup = %q", svc.AddrBackup)
	}
}
