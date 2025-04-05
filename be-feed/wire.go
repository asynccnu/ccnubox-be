//go:generate wire
//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-feed/cron"
	"github.com/asynccnu/ccnubox-be/be-feed/events"
	"github.com/asynccnu/ccnubox-be/be-feed/events/producer"
	"github.com/asynccnu/ccnubox-be/be-feed/grpc"
	"github.com/asynccnu/ccnubox-be/be-feed/ioc"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/google/wire"
)

func InitApp() App {
	wire.Build(

		grpc.NewFeedServiceServer,
		//feed服务
		service.NewPushService,
		service.NewFeedUserConfigService,
		service.NewMuxiOfficialMSGService,
		service.NewFeedEventService,
		//dao层
		dao.NewUserFeedConfigDAO,
		dao.NewFeedEventDAO,
		dao.NewUserFeedTokenDAO,
		dao.NewFeedFailEventDAO,
		//cache层一个
		cache.NewRedisFeedEventCache,
		//auto服务层三个
		cron.NewMuxiController,
		cron.NewCron,
		//event消费者控制服务
		events.NewFeedEventConsumerHandler,
		//push服务,consumer服务,producer服务
		producer.NewSaramaProducer,

		ioc.InitConsumers,
		ioc.InitDB,
		ioc.InitRedis,
		ioc.InitEtcdClient,
		ioc.InitLogger,
		ioc.InitKafka,
		ioc.InitJPushClient,
		ioc.InitGRPCxKratosServer,
		NewApp,
	)
	return App{}
}
