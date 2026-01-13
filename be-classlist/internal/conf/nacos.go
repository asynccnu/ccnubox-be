package conf

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
)

const (
	ClassList = "CCNUBOX_CLASSLIST_NACOS_DSN"
)

func InitBootstrap() *Bootstrap {
	return conf.InitConfig[Bootstrap](ClassList)
}
