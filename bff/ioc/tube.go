package ioc

import (
	"fmt"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/qiniu/api.v7/v7/storage"
)

type TubePolicies struct {
	defaultPolicy storage.PutPolicy
	officialSite  storage.PutPolicy
}

func InitTubePolicies(cfg *conf.ServerConf) *TubePolicies {
	return &TubePolicies{
		defaultPolicy: InitPutPolicy(cfg),
		officialSite:  InitOfficialSitePutPolicy(cfg),
	}
}

func InitPutPolicy(cfg *conf.ServerConf) storage.PutPolicy {
	return storage.PutPolicy{
		Scope:   cfg.Oss.BucketName,
		Expires: 60 * 60 * 24, // 一天过期
	}
}

func InitOfficialSitePutPolicy(cfg *conf.ServerConf) storage.PutPolicy {
	return storage.PutPolicy{
		Scope:   fmt.Sprintf("%s:%s%s", cfg.Oss.BucketName, cfg.Oss.BaseName, cfg.Oss.FileName),
		Expires: 60 * 60,
	}
}

func InitMac(cfg *conf.ServerConf) *qbox.Mac {
	return qbox.NewMac(cfg.Oss.AccessKey, cfg.Oss.SecretKey)
}
