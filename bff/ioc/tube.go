package ioc

import (
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/qiniu/api.v7/v7/storage"
	"github.com/spf13/viper"
)

func InitPutPolicy() storage.PutPolicy {
	return storage.PutPolicy{
		Scope:   viper.GetString("oss.bucketName"),
		Expires: 60 * 60 * 24, // 一天过期
	}
}

func InitMac() *qbox.Mac {
	type oss struct {
		AccessKey string `json:"accessKey"`
		SecretKey string `json:"secretKey"`
	}
	var cfg oss
	err := viper.UnmarshalKey("oss", &cfg)
	if err != nil {
		panic(err)
	}
	return qbox.NewMac(cfg.AccessKey, cfg.SecretKey)
}
