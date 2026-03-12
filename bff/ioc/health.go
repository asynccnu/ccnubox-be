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
		b_grpc.USER: client.InitClient(ecli, client.GetConf(cfg.Grpc, b_grpc.USER), cfg.Env, healthpb.NewHealthClient),
	}
}
