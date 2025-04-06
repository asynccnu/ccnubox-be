package pkg

import (
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/pkg/crawler"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(crawler.NewClassCrawler)
