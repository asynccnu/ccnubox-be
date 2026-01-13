package main

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/gin-gonic/gin"
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
	g        *gin.Engine
	cfg      *conf.HttpConf
}

func NewApp(
	g *gin.Engine,
	cfg *conf.ServerConf,
	shutdown func(ctx context.Context) error,
) *App {
	return &App{
		g:        g,
		cfg:      cfg.Http,
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
	addr := app.cfg.Addr
	err := app.g.Run(addr)
	if err != nil {
		return
	}
}
