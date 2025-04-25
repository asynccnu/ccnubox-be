//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/client"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/crawler"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/registry"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/server"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/service"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"os"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Registry, *conf.SchoolDay, *os.File, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet,
		data.ProviderSet,
		biz.ProviderSet,
		pkg.ProviderSet,
		registry.ProviderSet,
		service.ProviderSet,
		client.ProviderSet,
		newApp,
		wire.Bind(new(biz.ClassCrawler), new(*crawler.Crawler)),
		wire.Bind(new(biz.ClassInfoDBRepo), new(*data.ClassInfoDBRepo)),
		wire.Bind(new(biz.ClassInfoCacheRepo), new(*data.ClassInfoCacheRepo)),
		wire.Bind(new(biz.StudentAndCourseDBRepo), new(*data.StudentAndCourseDBRepo)),
		wire.Bind(new(biz.StudentAndCourseCacheRepo), new(*data.StudentAndCourseCacheRepo)),
		wire.Bind(new(service.ClassCtrl), new(*biz.ClassUsecase)),
		wire.Bind(new(biz.CCNUServiceProxy), new(*client.CCNUService)),
		wire.Bind(new(biz.ClassRepoProxy), new(*biz.ClassRepo)),
		wire.Bind(new(biz.JxbRepo), new(*data.JxbDBRepo)),
		wire.Bind(new(biz.Transaction), new(*data.Data)),
	))
}
