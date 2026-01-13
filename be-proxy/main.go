package main

import (
	"context"
	"fmt"
	"time"

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
	server   grpcx.Server
	shutdown func(ctx context.Context) error
}

func NewApp(server grpcx.Server,
	shutdown func(ctx context.Context) error,
) *App {
	return &App{
		server:   server,
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

	err := app.server.Serve()
	if err != nil {
		panic(err)
	}
}
