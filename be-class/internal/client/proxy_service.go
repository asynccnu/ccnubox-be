package client

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func InitProxyClient(r *etcd.Registry, cf *conf.Registry, logger log.Logger) (proxyv1.ProxyClient, error) {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(cf.Proxysvc),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(40*time.Second), //由于使用华师的服务,所以设置下超时时间最长为40s
		grpc.WithMiddleware(
			tracing.Client(),
			recovery.Recovery(),
		),
	)
	if err != nil {
		log.NewHelper(logger).WithContext(context.Background()).Errorw("kind", "grpc-client", "reason", "GRPC_CLIENT_INIT_ERROR", "err", err)
		return nil, err
	}
	return proxyv1.NewProxyClient(conn), nil
}
