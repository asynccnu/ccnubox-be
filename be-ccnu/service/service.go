package service

import (
	"context"
	"fmt"
)

// Crawler 描述 CCNUService 所需的完整外部系统能力。
// 它不会暴露 Passport、HTTP client、Cookie 或其他 crawler 内部实现细节。
type Crawler interface {
	LoginUnderGraduate(ctx context.Context, studentID, password string) (bool, error)
	LoginPostGraduate(ctx context.Context, studentID, password string) (bool, error)

	GetUnderGraduateCookie(ctx context.Context, studentID, password string) (string, error)
	GetPostGraduateCookie(ctx context.Context, studentID, password string) (string, error)

	GetLibrarySeatToken(ctx context.Context, studentID, password string) (string, error)
	GetLibraryDiscussionToken(ctx context.Context, studentID, password string) (string, error)

	CheckLibrarySeatToken(ctx context.Context, token string) (bool, error)
	CheckLibraryDiscussionToken(ctx context.Context, token string) (bool, error)
}

// CrawlerFactory 为每次 service 操作创建新的 crawler。
// 单次 crawler 操作可以持有本次请求独立的 HTTP client 和 CookieJar。
type CrawlerFactory interface {
	New() Crawler
}

type CrawlerErrorKind uint8

const (
	CrawlerInvalidCredential CrawlerErrorKind = iota + 1
	CrawlerAccountInitializationRequired
	CrawlerUpstreamFailure
)

// CrawlerError 是 service 与完整 crawler 实现之间唯一共享的错误语义。
// Cause 保留原始技术错误。
type CrawlerError struct {
	Kind  CrawlerErrorKind
	Cause error
}

func NewCrawlerError(kind CrawlerErrorKind, cause error) *CrawlerError {
	return &CrawlerError{Kind: kind, Cause: cause}
}

func (e *CrawlerError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("crawler failure kind %d", e.Kind)
	}
	return e.Cause.Error()
}

func (e *CrawlerError) Unwrap() error {
	return e.Cause
}
