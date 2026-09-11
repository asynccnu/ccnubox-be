package ioc

import (
	"fmt"
	"net/http"

	"github.com/asynccnu/ccnubox-be/be-library/conf"
	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/service"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	bgrpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

func InitReminderCrawler(httpClient *http.Client, cfg *conf.ServerConf, metrics *metricsx.Metrics) crawler.ReminderCrawler {
	reminder := cfg.Reminder()
	return crawler.NewReminderCrawler(httpClient, reminder.RequestTimeout, reminder.HistoryPageSize, reminder.HistoryLookbackDays, reminder.UpstreamQPS, metrics.Library)
}

func InitReminderFeedClient(etcdClient *etcdv3.Client, cfg *conf.InfraConf, serverCfg *conf.ServerConf) feedv1.FeedServiceClient {
	if !serverCfg.Reminder().Enabled {
		return nil
	}
	if cfg == nil || cfg.Grpc == nil {
		panic("library reminder is enabled but grpc configuration is missing")
	}
	feedCfg, ok := (*cfg.Grpc)[bgrpc.FEED]
	if !ok || feedCfg == nil {
		panic(fmt.Sprintf("library reminder is enabled but Feed grpc configuration is missing"))
	}
	return client.InitClient(etcdClient, feedCfg, cfg.Env, feedv1.NewFeedServiceClient)
}

// InitReminderFeedGateway 保留 logger 参数，使 Wire 与旧版爬虫共享相同的运行时依赖，
// 同时便于后续接入指标。
func InitReminderFeedGateway(client feedv1.FeedServiceClient, _ logger.Logger) service.FeedGateway {
	return service.NewFeedGateway(client)
}
