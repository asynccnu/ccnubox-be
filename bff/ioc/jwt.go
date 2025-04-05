package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// InitJwtHandler 初始化 JWT 处理程序，并返回一个 ijwt.Handler
// 参数 cmd 是 redis.Cmdable 接口，用于与 Redis 进行交互
func InitJwtHandler(cmd redis.Cmdable) ijwt.Handler {
	// 定义一个配置结构体，用于存储从配置文件中读取的 JWT 配置,
	//包括用于生成长短token的两个配置,长token保存时间较长,
	//使用长token获取短token，短token进行身份验证,可以有效加强安全性,防止用户账号被盗用。
	type Config struct {
		JwtKey     string `yaml:"jwtKey"`
		RefreshKey string `yaml:"refreshKey"`
	}

	var cfg Config

	// 从配置文件中读取 JWT 配置，并将其解码到 Config 结构体中
	err := viper.UnmarshalKey("jwt", &cfg)
	if err != nil {
		// 如果读取配置失败，则抛出一个 panic 错误
		panic(err)
	}

	// 返回一个新的 RedisJWTHandler 实例
	// 传递 Redis 命令接口和配置中的 JwtKey 和 RefreshKey
	return ijwt.NewRedisJWTHandler(cmd, cfg.JwtKey, cfg.RefreshKey)
}
