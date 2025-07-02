//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/client"
	"github.com/asynccnu/ccnubox-be/be-library/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data"
	"github.com/asynccnu/ccnubox-be/be-library/internal/registry"
	"github.com/asynccnu/ccnubox-be/be-library/internal/server"
	"github.com/asynccnu/ccnubox-be/be-library/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Registry, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		client.ProviderSet,
		registry.ProviderSet,
		newApp,
		// 更接头化的解耦
		wire.Bind(new(biz.DelayQueue), new(*data.DelayKafka)),
	))
}
