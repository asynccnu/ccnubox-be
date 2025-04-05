package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
	"github.com/spf13/viper"
)

func InitJPushClient() jpush.PushClient {
	//配置获取
	type Config struct {
		AppKey       string `yaml:"appKey"`
		MasterSecret string `yaml:"masterSecret"`
	}

	var cfg Config
	err := viper.UnmarshalKey("jpushConfig", &cfg)
	if err != nil {
		return nil
	}

	client := jpush.NewJPushClient(cfg.AppKey, cfg.MasterSecret)

	return client
}
