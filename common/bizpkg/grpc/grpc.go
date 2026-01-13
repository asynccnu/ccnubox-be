package grpc

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/identity"
)

const (
	CLASSLIST = "classlist"
	USER      = "user"
	GRADE     = "grade"
	LIBRARY   = "library"
	CONTENT   = "content"
	ELECPRICE = "elecprice"
	CCNU      = "ccnu"
	COUNTER   = "counter"
	FEED      = "feed"
	PROXY     = "proxy"
	CLASSS    = "class"
)

func GetNamePrefix(env *conf.Env, name string) string {
	// 仅在开发环境增加个人标识前缀，其余环境要上线所以不能加，否则会因为容器名称不一致导致无法正常通信
	name = env.String() + "/" + name
	if env.IsDev() {
		name = identity.GetIdentity() + "/" + name
	}

	return name
}
