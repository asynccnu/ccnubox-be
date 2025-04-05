// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-static/grpc"
	"github.com/asynccnu/ccnubox-be/be-static/ioc"
	"github.com/asynccnu/ccnubox-be/be-static/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-static/repository"
	"github.com/asynccnu/ccnubox-be/be-static/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-static/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-static/service"
)

// Injectors from wire.go:

func InitGRPCServer() grpcx.Server {
	database := ioc.InitDB()
	staticDAO := dao.NewMongoDBStaticDAO(database)
	cmdable := ioc.InitRedis()
	staticCache := cache.NewRedisStaticCache(cmdable)
	logger := ioc.InitLogger()
	staticRepository := repository.NewCachedStaticRepository(staticDAO, staticCache, logger)
	staticService := service.NewStaticService(staticRepository)
	staticServiceServer := grpc.NewStaticServiceServer(staticService)
	client := ioc.InitEtcdClient()
	server := ioc.InitGRPCxKratosServer(staticServiceServer, client, logger)
	return server
}
