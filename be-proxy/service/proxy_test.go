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
