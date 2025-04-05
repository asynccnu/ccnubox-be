package main

import (
	"github.com/asynccnu/ccnubox-be/be-feed/cron"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/saramax"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
	app := InitApp()
	app.Start()
}

func initViper() {
	cfile := pflag.String("config", "config/config.yaml", "配置文件路径")
	pflag.Parse()

	viper.SetConfigType("yaml")
	viper.SetConfigFile(*cfile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
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
