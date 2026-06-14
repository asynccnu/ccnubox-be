package ioc

import (
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
)

func InitMetrics() *metricsx.Metrics {
	return metricsx.New("ccnubox")
}
