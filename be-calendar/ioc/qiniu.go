package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-calendar/conf"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/qiniu"
)

func InitQiniu(cfg *conf.TransConf) qiniu.QiniuClient {
	return qiniu.NewQiniuService(cfg.QiNiu.AccessKey, cfg.QiNiu.SecretKey, cfg.QiNiu.Bucket, cfg.QiNiu.Domain, cfg.QiNiu.BaseName)
}
