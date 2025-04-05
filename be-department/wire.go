//go:build wireinject
// +build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-department/grpc"
	"github.com/asynccnu/ccnubox-be/be-department/ioc"
	"github.com/asynccnu/ccnubox-be/be-department/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-department/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-department/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-department/service"
	"github.com/google/wire"
)

func InitGRPCServer() grpcx.Server {
	wire.Build(
		ioc.InitGRPCxKratosServer,
		grpc.NewDepartmentServiceServer,
		service.NewDepartmentService,
		cache.NewRedisDepartmentCache,
		dao.NewMysqlDepartmentDAO,
		// 第三方
		ioc.InitDB,
		ioc.InitRedis,
		ioc.InitLogger,
		ioc.InitEtcdClient,
	)
	return grpcx.Server(nil)
}
