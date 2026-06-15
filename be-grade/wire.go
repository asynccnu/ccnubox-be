//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-grade/conf"
	"github.com/asynccnu/ccnubox-be/be-grade/events"
	"github.com/asynccnu/ccnubox-be/be-grade/events/producer"
	"github.com/asynccnu/ccnubox-be/be-grade/grpc"
	"github.com/asynccnu/ccnubox-be/be-grade/ioc"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(
		conf.InitInfraConfig,
		conf.InitServerConf,
		events.NewGradeDetailEventConsumerHandler,
		producer.NewInstrumentedSaramaProducer,
		grpc.NewGradeGrpcService,
		service.NewGradeService,
		service.NewRankService,
		dao.NewGradeDAO,
		dao.NewRankDAO,
		ioc.InitEtcdClient,
		ioc.InitOTel,
		ioc.InitDB,
		ioc.InitLogger,
		ioc.InitGRPCxKratosServer,
		ioc.InitUserClient,
		ioc.InitProxyClient,
		ioc.InitHttpProxyClient,
		ioc.InitClassListClient,
		ioc.InitKafka,
		ioc.InitMetrics,
		ioc.InitMetricsServer,
		ioc.InitConsumers,
		NewApp,
	)
	return App{}
}
