package data

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"

	com_cfg "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/otel"
)

func InitOTel(cfg *conf.OtelConfig) func(ctx context.Context) error {
	otelCfg := &com_cfg.OtelConf{
		ServiceName:    cfg.GetServiceName(),
		ServiceVersion: cfg.GetServiceVersion(),
		Endpoint:       cfg.GetEndpoint(),
	}

	return otel.InitOTel(otelCfg)
}
