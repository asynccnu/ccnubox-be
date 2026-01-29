package main

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-content/cron"
	"github.com/asynccnu/ccnubox-be/common/pkg/grpcx"
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
	server   grpcx.Server
	crons    []cron.Cron
}

func NewApp(server grpcx.Server,
	crons []cron.Cron, shutdown func(ctx context.Context) error) *App {
	return &App{
		server:   server,
		crons:    crons,
		shutdown: shutdown,
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

	err := app.server.Serve()
	if err != nil {
		return
	}
}
