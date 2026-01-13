package client

import (
	"context"
	"fmt"
	"time"

	b_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	b_grpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	k_grpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/wire"
	"google.golang.org/grpc"
)

var ProviderSet = wire.NewSet(NewEnv, NewUserSvc, InitProxyClient)

// TODO 现在是通过强制手动写死的方式实现的根据环境适配注册中心的注册，需要改成通用的方案
func InitClient(r *etcd.Registry, name string, env *b_conf.Env) (*grpc.ClientConn, error) {

	name = b_grpc.GetNamePrefix(env, name)

	conn, err := k_grpc.DialInsecure(
		context.Background(),
		k_grpc.WithEndpoint(fmt.Sprintf("discovery:///%s", name)),
		k_grpc.WithDiscovery(r),
		k_grpc.WithTimeout(40*time.Second), //由于使用华师的服务,所以设置下超时时间最长为40s
		k_grpc.WithMiddleware(
			tracing.Client(),
			recovery.Recovery(),
		),
	)
	return conn, err
}

func NewEnv(env string) *b_conf.Env {
	confEnv := b_conf.Env(env)
	return &confEnv
}
