package service

import (
	"context"
	"time"

	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

type CCNUService interface {
	LoginCCNU(ctx context.Context, studentId string, password string) (bool, error)
	GetXKCookie(ctx context.Context, studentId string, password string, system tool.XKSystem) (string, error)
	CheckXKCookie(ctx context.Context, cookie string, system tool.XKSystem) (bool, error)
	GetLibraryToken(ctx context.Context, studentId, password string, service ccnuv1.LIBRARY_TYPE) (string, error)
	CheckLibraryToken(ctx context.Context, token string, service ccnuv1.LIBRARY_TYPE) (bool, error)
}

type ccnuService struct {
	timeout time.Duration
	l       logger.Logger
	p       proxy.Client
	secret  string
}

func NewCCNUService(l logger.Logger, p proxy.Client, secret string) CCNUService {
	return &ccnuService{
		timeout: time.Minute * 2,
		l:       l,
		p:       p,
		secret:  secret,
	}
}
