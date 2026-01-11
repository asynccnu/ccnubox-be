package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
)

func InitJPushClient(cfg *conf.TransConf) jpush.PushClient {
	client := jpush.NewJPushClient(cfg.JPush.AppKey, cfg.JPush.MasterSecret)

	return client
}
