package main

import (
	"github.com/asynccnu/ccnubox-be/be-calendar/cron"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
)

func main() {
	app := InitApp()
	app.Start()
}

type App struct {
	server grpcx.Server
	crons  []cron.Cron
}

func NewApp(server grpcx.Server,
	crons []cron.Cron) App {
	return App{
		server: server,
		crons:  crons,
	}
}

func (a *App) Start() {
	for _, c := range a.crons {
		c.StartCronTask()
	}

	err := a.server.Serve()
	if err != nil {
		return
	}
}
