package service

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-ccnu/pkg/logger"
	"time"
)

type CCNUService interface {
	Login(ctx context.Context, studentId string, password string) (bool, error)
	GetCCNUCookie(ctx context.Context, studentId string, password string) (string, error)
}

type ccnuService struct {
	timeout time.Duration
	l       logger.Logger
}

func NewCCNUService(l logger.Logger) CCNUService {
	return &ccnuService{
		timeout: time.Minute * 2,
		l:       l,
	}
}
