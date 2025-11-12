package ioc

import (
	"fmt"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/qiniu/api.v7/v7/storage"
	"github.com/spf13/viper"
)

type TubePolicies struct {
	defaultPolicy storage.PutPolicy
	officialSite  storage.PutPolicy
}

func InitTubePolicies() *TubePolicies {
	return &TubePolicies{
		defaultPolicy: InitPutPolicy(),
		officialSite:  InitOfficialSitePutPolicy(),
	}
}

func InitPutPolicy() storage.PutPolicy {
	return storage.PutPolicy{
		Scope:   viper.GetString("oss.bucketName"),
		Expires: 60 * 60 * 24, // 一天过期
	}
}

func InitOfficialSitePutPolicy() storage.PutPolicy {
	return storage.PutPolicy{
		Scope:   fmt.Sprintf("%s:%s%s", viper.GetString("oss.bucketName"), viper.GetString("oss.baseName"), viper.GetString("oss.appName")),
		Expires: 60 * 60,
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
