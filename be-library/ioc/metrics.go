package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-library/conf"
	bgrpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
)

func InitMetrics() *metricsx.Metrics {
	return metricsx.New("ccnubox")
}

func InitMetricsServer(cfg *conf.InfraConf) *metricsx.Server {
	if cfg == nil || cfg.Grpc == nil {
		return metricsx.NewServer("")
	}
	grpcCfg := (*cfg.Grpc)[bgrpc.LIBRARY]
	if grpcCfg == nil {
		return metricsx.NewServer("")
	}
	return metricsx.NewServerFromGRPCAddr(grpcCfg.Addr)
}
