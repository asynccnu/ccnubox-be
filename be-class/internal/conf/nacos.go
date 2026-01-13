package conf

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
)

const (
	Class = "CCNUBOX_CLASS_NACOS_DSN"
)

// TODO kratos使用proto生成的配置可能会有兼容性问题,建议后续改成手动定义配置,而不是利用proto生成
func InitBootstrap() *Bootstrap {
	return conf.InitConfig[Bootstrap](Class)
}
