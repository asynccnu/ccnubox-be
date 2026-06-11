package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
)

func InitMetrics(cfg *conf.ServerConf) *metricsx.Metrics {
	return metricsx.New("ccnubox")
}