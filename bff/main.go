package main

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/bff/ioc"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := conf.InitTransConfig()
	if cfg == nil {
		panic("transCfg is nil")
	}
	// 初始化 OTel 并注册优雅关闭
	shutdown := ioc.InitOTel(cfg)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			panic(fmt.Sprintln("OTel shutdown error:", err))
		}
	}()

	app := InitApp()
	app.Start(cfg)
}

type App struct {
	g *gin.Engine
}

func NewApp(g *gin.Engine) *App {
	return &App{g: g}
}

func (app *App) Start(cfg *conf.TransConf) {
	addr := cfg.Http.Addr
	err := app.g.Run(addr)
	if err != nil {
		return
	}
}
