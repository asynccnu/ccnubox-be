package client

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	b_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
)

func InitProxyClient(r *etcd.Registry, cf *conf.Registry, logger log.Logger, env *b_conf.Env) (proxyv1.ProxyClient, error) {
	conn, err := InitClient(r, cf.Proxysvc, env)
	if err != nil {
		log.NewHelper(logger).WithContext(context.Background()).Errorw("kind", "grpc-client", "reason", "GRPC_CLIENT_INIT_ERROR", "err", err)
		return nil, err
	}
	return proxyv1.NewProxyClient(conn), nil
}
