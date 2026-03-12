package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	b_grpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"
	clientv3 "go.etcd.io/etcd/client/v3"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func InitHealthClient(ecli *clientv3.Client, cfg *conf.InfraConf) map[string]healthpb.HealthClient {
	return map[string]healthpb.HealthClient{
		b_grpc.CCNU: client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.CCNU), cfg.Env, healthpb.NewHealthClient),
		//b_grpc.CLASSS:    client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.CLASSS), cfg.Env, healthpb.NewHealthClient),
		//b_grpc.CLASSLIST: client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.CLASSLIST), cfg.Env, healthpb.NewHealthClient),
		b_grpc.CONTENT:   client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.CONTENT), cfg.Env, healthpb.NewHealthClient),
		b_grpc.COUNTER:   client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.COUNTER), cfg.Env, healthpb.NewHealthClient),
		b_grpc.ELECPRICE: client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.ELECPRICE), cfg.Env, healthpb.NewHealthClient),
		b_grpc.FEED:      client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.FEED), cfg.Env, healthpb.NewHealthClient),
		b_grpc.GRADE:     client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.GRADE), cfg.Env, healthpb.NewHealthClient),
		//b_grpc.LIBRARY:   client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.LIBRARY), cfg.Env, healthpb.NewHealthClient),
		b_grpc.PROXY: client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.PROXY), cfg.Env, healthpb.NewHealthClient),
		b_grpc.USER:  client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.USER), cfg.Env, healthpb.NewHealthClient),
	}
}
