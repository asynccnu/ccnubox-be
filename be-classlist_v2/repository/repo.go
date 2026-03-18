package repo

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewClassRepo,
	NewJxbRepo,
	NewRefreshLogRepo,
)
