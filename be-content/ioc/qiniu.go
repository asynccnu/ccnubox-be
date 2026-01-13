package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-content/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/qiniu"
)

func InitQiniu(cfg *conf.ServerConf) qiniu.QiniuClient {
	qu := cfg.Qiniu
	return qiniu.NewQiniuService(qu.AccessKey, qu.SecretKey, qu.Bucket, qu.Domain, qu.BaseName)
}
