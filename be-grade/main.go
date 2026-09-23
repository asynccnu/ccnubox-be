package main

import (
	"context"
	"fmt"
	"log"
	"time"

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
	server    grpcx.Server
	metrics   *metricsx.Server
	consumers []saramax.Consumer
	shutdown  func(ctx context.Context) error
}

func NewApp(
	server grpcx.Server,
	metrics *metricsx.Server,
	consumers []saramax.Consumer,
	shutdown func(ctx context.Context) error,
) App {
	return App{
		server:    server,
		metrics:   metrics,
		consumers: consumers,
		shutdown:  shutdown,
	}
}

func (app *App) Start() {
	defer func() {
		// 消费者可能正卡在一批消息上（单批最多上百条且要逐条处理），
		// 用独立超时兜住停机，超时后保留未确认位点直接退出。
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
		if !saramax.StopConsumers(stopCtx, app.consumers) {
			log.Printf("停止消费者超时，继续退出进程")
		}
		stopCancel()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.shutdown(ctx); err != nil {
			panic(fmt.Sprintln("shutdown error:", err))
		}
		if err := app.metrics.Close(); err != nil {
			panic(fmt.Sprintln("metrics shutdown error:", err))
		}
	}()

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
