package main

import (
	"github.com/asynccnu/ccnubox-be/be-elecprice/cron"
	"github.com/asynccnu/ccnubox-be/be-elecprice/pkg/grpcx"
	//
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
