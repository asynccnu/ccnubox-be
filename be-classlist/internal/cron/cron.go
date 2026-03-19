package cron

import "github.com/google/wire"

type Cron interface {
	StartCronTask()
}

var ProviderSet = wire.NewSet(NewClassListController)
