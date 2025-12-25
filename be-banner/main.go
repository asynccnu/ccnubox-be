package main

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
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
