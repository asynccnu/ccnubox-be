package classLog

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewClogger)

type Clogger interface {
	Warnw(keyvals ...interface{})
	Errorw(keyvals ...interface{})
	Warn(a ...interface{})
}

func NewClogger(l log.Logger) *log.Helper {
	return log.NewHelper(l)
}
