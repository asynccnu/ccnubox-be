package service

import (
	"context"
	proxyv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/proxy/v1"
	"time"

	"github.com/asynccnu/ccnubox-be/be-ccnu/pkg/logger"
)

type CCNUService interface {
	LoginCCNU(ctx context.Context, studentId string, password string) (bool, error)
	GetXKCookie(ctx context.Context, studentId string, password string) (string, error)
	GetLibraryCookie(ctx context.Context, studentId, password string) (string, error)
}

type ccnuService struct {
	timeout time.Duration
	l       logger.Logger
	p       proxyv1.ProxyClient
}

func NewCCNUService(l logger.Logger, p proxyv1.ProxyClient) CCNUService {
	return &ccnuService{
		timeout: time.Minute * 2,
		l:       l,
		p:       p,
	}
}
