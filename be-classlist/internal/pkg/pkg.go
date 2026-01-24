package pkg

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/crawler"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(crawler.NewClassCrawler, crawler.NewClassCrawler2, crawler.NewClassCrawler3, crawler.NewProxyGetter)
