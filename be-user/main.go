package main

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-user/conf"
	"github.com/asynccnu/ccnubox-be/be-user/ioc"
)

func main() {
	transCfg := conf.InitTransConfig()
	if transCfg == nil {
		panic("transCfg is nil")
	}
	// 初始化 OTel 并注册优雅关闭
	shutdown := ioc.InitOTel(transCfg)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			panic(fmt.Sprintln("OTel shutdown error:", err))
		}
	}()

	server := InitGRPCServer()
	err := server.Serve()
	if err != nil {
		panic(err)
	}
}
