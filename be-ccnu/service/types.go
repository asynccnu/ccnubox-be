package service

import (
	"context"
	"time"

	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type CCNUService interface {
	LoginCCNU(ctx context.Context, studentId string, password string) (bool, error)
	GetXKCookie(ctx context.Context, studentId string, password string, tpe ...string) (string, error) // 传入可变参数, 代码侵入性低
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
