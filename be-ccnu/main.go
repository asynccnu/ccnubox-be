package main

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-ccnu/ioc"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()

	// 初始化 OTel 并注册优雅关闭
	shutdown := ioc.InitOTel()
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
