package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-user/conf"
	"github.com/asynccnu/ccnubox-be/be-user/grpc"
	b_grpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/server"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitGRPCxKratosServer(grpcServer *grpc.UserServiceServer, ecli *clientv3.Client, l logger.Logger, cfg *conf.InfraConf) grpcx.Server {
	return server.InitGRPCxKratosServer(grpcServer, ecli, l, (*cfg.Grpc)[b_grpc.USER], cfg.Env)
}
