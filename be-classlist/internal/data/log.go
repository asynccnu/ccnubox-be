package data

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/adapter"

	com_cfg "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	bizlog "github.com/asynccnu/ccnubox-be/common/bizpkg/log"
	klog "github.com/go-kratos/kratos/v2/log"
	glog "gorm.io/gorm/logger"
)

// TODO 这里没有 nacos 自动将配置转换回 common 格式的呢？
func NewLogger(cfg *conf.ZapLogConfigs) logger.Logger {
	commonCfg := &com_cfg.LogConf{
		Path:       cfg.LogPath,
		MaxSize:    int(cfg.LogFileMaxSize),
		MaxBackups: int(cfg.LogFileMaxBackups),
		MaxAge:     int(cfg.LogMaxAge),
		Compress:   cfg.LogCompress,
	}

	return bizlog.InitLogger(commonCfg, 3)
}

func NewKratosLogger(l logger.Logger) klog.Logger {
	return adapter.NewKratosLogger(l)
}

func NewGromLogger(l logger.Logger) glog.Interface {
	return adapter.NewGormLogger(l)
}
