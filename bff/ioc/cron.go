package ioc

import (
	"github.com/asynccnu/ccnubox-be/common/pkg/cronx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

func InitCronxManager(l logger.Logger) *cronx.Manager {
	return cronx.NewManager(l)
}
