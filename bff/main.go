package main

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag" // 导入 pflag 包，用于命令行参数解析
	"github.com/spf13/viper" // 导入 viper 包，用于配置文件解析
)

func main() {
	initViper() // 初始化 viper 以读取配置文件
	app := InitApp()
	app.Start()
}

func initViper() {
	// 根据配置文件初始化 viper,viper能够把配置文件按照键值对的方式加载到内存中去。
	cfile := pflag.String("config", "config/config.yaml", "配置文件路径") // 定义命令行参数，用于指定配置文件路径，默认为 "config/config.yaml"
	pflag.Parse()                                                   // 解析命令行参数
	viper.SetConfigType("yaml")                                     // 设置配置文件类型为 YAML
	viper.SetConfigFile(*cfile)                                     // 设置配置文件路径为解析后的命令行参数值
	err := viper.ReadInConfig()                                     // 读取配置文件
	if err != nil {                                                 // 如果读取配置文件时发生错误，则抛出异常
		panic(err)
	}
}

type App struct {
	g *gin.Engine
}

func NewApp(g *gin.Engine) *App {
	return &App{g: g}
}

func (app *App) Start() {
	addr := viper.GetString("http.addr")
	err := app.g.Run(addr)
	if err != nil {
		return
	}
}
