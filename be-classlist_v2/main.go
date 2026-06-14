package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	app, cleanup, err := InitApp()
	if err != nil {
		panic(err)
	}
	defer cleanup()
	app.Start()
}

type App struct {
	server   grpcx.Server
	metrics  *metricsx.Server
	shutdown func(ctx context.Context) error
}

func NewApp(
	server grpcx.Server,
	metrics *metricsx.Server,
	shutdown func(ctx context.Context) error,
) *App {
	return &App{
		server:   server,
		metrics:  metrics,
		shutdown: shutdown,
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
