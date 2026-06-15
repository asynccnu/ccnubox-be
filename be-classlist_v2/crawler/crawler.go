package crawler

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewClassCrawler,
	NewClassCrawler2,
	NewClassCrawler3,
)
