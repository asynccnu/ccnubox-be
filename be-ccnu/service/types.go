package service

import (
	"context"

	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type CCNUService interface {
	LoginCCNU(ctx context.Context, studentId string, password string) (bool, error)
	GetXKCookie(ctx context.Context, studentId string, password string, tpe ...string) (string, error) // 传入可变参数, 代码侵入性低
	GetLibraryToken(ctx context.Context, studentId, password string, service ccnuv1.LIBRARY_TYPE) (string, error)
	CheckLibraryToken(ctx context.Context, token string, service ccnuv1.LIBRARY_TYPE) (bool, error)
}

type ccnuService struct {
	l       logger.Logger
	factory CrawlerFactory
}

func NewCCNUService(l logger.Logger, factory CrawlerFactory) CCNUService {
	return &ccnuService{
		l:       l,
		factory: factory,
	}
}
