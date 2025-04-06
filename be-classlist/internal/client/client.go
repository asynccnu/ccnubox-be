package client

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewClient, NewCCNUService)
