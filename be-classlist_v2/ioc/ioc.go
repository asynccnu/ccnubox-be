package ioc

import (
	"github.com/google/wire"
)

var Provider = wire.NewSet(
	initLogger,
)
