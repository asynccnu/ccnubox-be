package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/cron"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	app := InitApp()
	app.Start()
}

type App struct {
	shutdown func(ctx context.Context) error

	server    grpcx.Server
	metrics   *metricsx.Server
	consumers []saramax.Consumer
	crons     []cron.Cron
}

func NewApp(
	server grpcx.Server,
	metrics *metricsx.Server,
	crons []cron.Cron,
	consumers []saramax.Consumer,
	shutdown func(ctx context.Context) error,
) *App {
	return &App{
		shutdown:  shutdown,
		server:    server,
		metrics:   metrics,
		crons:     crons,
		consumers: consumers,
	}
}

func (app *App) Start() {
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.shutdown(ctx); err != nil {
			panic(fmt.Sprintln("shutdown error:", err))
		}
		if err := app.metrics.Close(); err != nil {
			panic(fmt.Sprintln("metrics shutdown error:", err))
		}
	}()

	for _, c := range app.crons {
		c.StartCronTask()
	}

	for _, c := range app.consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}

	go func() {
		// metrics 是辅助通道, 失败仅记录, 不拖垮主服务。
		if err := app.metrics.Serve(); err != nil {
			log.Printf("metrics server exit: addr=%s err=%v", app.metrics.Addr(), err)
		}
	}()

	err := app.server.Serve()
	if err != nil {
		panic(err)
	}
}
