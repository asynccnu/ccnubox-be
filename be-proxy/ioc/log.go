package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-proxy/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/log"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

func InitLogger(cfg *conf.ServerConf) logger.Logger {
	return log.InitLogger(cfg.Log, 3)
}
