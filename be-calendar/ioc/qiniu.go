package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/qiniu"
	"github.com/spf13/viper"
)

func InitQiniu() qiniu.QiniuClient {
	var cfg struct {
		AccessKey string `yaml:"accessKey"`
		SecretKey string `yaml:"secretKey"`
		Bucket    string `yaml:"bucket"`
		Domain    string `yaml:"domain"`
		BaseName  string `yaml:"baseName"`
	}

	err := viper.UnmarshalKey("qiniu", &cfg)
	if err != nil {
		panic(err)
	}

	return qiniu.NewQiniuService(cfg.AccessKey, cfg.SecretKey, cfg.Bucket, cfg.Domain, cfg.BaseName)
}
