//go:build wireinject

package main

import (
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/usecase"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/client"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/crawler"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/events"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/grpc"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/ioc"
	repo "github.com/asynccnu/ccnubox-be/be-classlist_v2/repository"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/service"
	"github.com/google/wire"
)

func InitApp() (*App, func(), error) {
	wire.Build(
		NewApp,
		usecase.ProviderSet,
		conf.ProviderSet,
		crawler.ProviderSet,
		events.ProviderSet,
		grpc.ProviderSet,
		ioc.ProviderSet,
		cache.ProviderSet,
		dao.ProviderSet,
		repo.ProviderSet,
		service.ProviderSet,
		client.ProviderSet,
		wire.Bind(new(biz.ClassCrawler), new(*crawler.Crawler3)),
	)
	return nil, nil, nil
}
