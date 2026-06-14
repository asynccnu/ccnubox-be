package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	b_grpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
)

func InitMetrics() *metricsx.Metrics {
	return metricsx.New("ccnubox")
}

// InitMetricsServer 基于 gRPC 监听地址派生 metrics HTTP 监听地址,
// 不依赖额外 yaml 字段, 端口 = gRPC 端口 + 1000。
func InitMetricsServer(cfg *conf.InfraConf) *metricsx.Server {
	grpcCfg := (*cfg.Grpc)[b_grpc.CLASSLIST]
	if grpcCfg == nil {
		return metricsx.NewServer("")
	}
	return metricsx.NewServerFromGRPCAddr(grpcCfg.Addr)
}
