package pkg

import (
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/pkg/crawler"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(crawler.NewClassCrawler)
