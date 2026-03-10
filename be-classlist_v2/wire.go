//go:build wireinject

package main

import (
	"github.com/google/wire"
)

func InitApp() *App {
	wire.Build(
		NewApp,
	)
	return &App{}
}
