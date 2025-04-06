package classLog

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewClogger)

type Clogger interface {
	Error(a ...interface{})
	Errorw(keyvals ...interface{})
	Errorf(format string, a ...interface{})
	Warn(a ...interface{})
	Warnw(keyvals ...interface{})
	Warnf(format string, a ...interface{})
	Infof(format string, a ...interface{})
}

func NewClogger(l log.Logger) *log.Helper {
	return log.NewHelper(l)
}
