package main

import (
	"github.com/asynccnu/ccnubox-be/be-feed/cron"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/saramax"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
)

func main() {
	app := InitApp()
	app.Start()
}

type App struct {
	server    grpcx.Server
	consumers []saramax.Consumer
	crons     []cron.Cron
}

func NewApp(server grpcx.Server,
	crons []cron.Cron,
	consumers []saramax.Consumer,
) App {
	return App{
		server:    server,
		crons:     crons,
		consumers: consumers,
	}
}

func (a *App) Start() {
	for _, c := range a.crons {
		c.StartCronTask()
	}

	//启动所有的消费者,但是这里实际上只注入了一个消费者
	for _, c := range a.consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}

	err := a.server.Serve()
	if err != nil {
		panic(err)
	}

}
