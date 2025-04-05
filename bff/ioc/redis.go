package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	//redis配置
	type Config struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
	}
	var cfg Config
	//获取redis配置具体信息
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(err)
	}
	//初始化一个redis
	return redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password})
}
