package main

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/cron"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/joho/godotenv"
)

func init() {
	// 预加载.env文件,用于本地开发
	_ = godotenv.Load()
}
func main() {
	app := InitApp()
	app.Start()
}

type App struct {
	shutdown func(ctx context.Context) error

	server    grpcx.Server
	consumers []saramax.Consumer
	crons     []cron.Cron
}

func NewApp(server grpcx.Server,
	crons []cron.Cron,
	consumers []saramax.Consumer,
	shutdown func(ctx context.Context) error,
) *App {
	return &App{
		shutdown:  shutdown,
		server:    server,
		crons:     crons,
		consumers: consumers,
	}
}

func (app *App) Start() {
	// 优雅关闭
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.shutdown(ctx); err != nil {
			panic(fmt.Sprintln("shutdown error:", err))
		}
	}()

	for _, c := range app.crons {
		c.StartCronTask()
	}

	//启动所有的消费者,但是这里实际上只注入了一个消费者
	for _, c := range app.consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}

	err := app.server.Serve()
	if err != nil {
		panic(err)
	}

}
