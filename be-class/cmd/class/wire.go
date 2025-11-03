//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/asynccnu/ccnubox-be/be-class/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-class/internal/client"
	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-class/internal/data"
	"github.com/asynccnu/ccnubox-be/be-class/internal/lock"
	"github.com/asynccnu/ccnubox-be/be-class/internal/registry"
	"github.com/asynccnu/ccnubox-be/be-class/internal/server"
	"github.com/asynccnu/ccnubox-be/be-class/internal/service"
	"github.com/asynccnu/ccnubox-be/be-class/internal/timedTask"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Registry, log.Logger) (*APP, func(), error) {
	panic(wire.Build(server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		registry.ProviderSet,
		client.ProviderSet,
		timedTask.ProviderSet,
		lock.ProviderSet,
		wire.Bind(new(biz.EsProxy), new(*data.ClassData)),
		wire.Bind(new(biz.ClassListService), new(*client.ClassListService)),
		wire.Bind(new(biz.FreeClassRoomData), new(*data.FreeClassroomData)),
		wire.Bind(new(biz.ClassData), new(*data.ClassData)),
		wire.Bind(new(biz.CookieClient), new(*client.CookieSvc)),
		wire.Bind(new(biz.Cache), new(*data.Cache)),
		wire.Bind(new(service.ClassInfoProxy), new(*biz.ClassServiceUserCase)),
		wire.Bind(new(service.FreeClassRoomSaver), new(*biz.FreeClassroomBiz)),
		wire.Bind(new(service.FreeClassroomSearcher), new(*biz.FreeClassroomBiz)),
		NewApp,
		newApp))
}
