package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/grpc/client"

	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitGradeClient(ecli *clientv3.Client, cfg *conf.InfraConf) gradev1.GradeServiceClient {
	return client.InitGrade(ecli, cfg.Grpc, cfg.Env)

}
