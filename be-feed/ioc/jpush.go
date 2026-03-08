package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
)

func InitJPushClient(cfg *conf.ServerConf) jpush.PushClient {
	client := jpush.NewJPushClient(&jpush.JPushConfig{
		AppKey:       cfg.JPush.AppKey,
		MasterSecret: cfg.JPush.MasterSecret,
		HUAWEI: struct {
			Category string `json:"category"`
		}{
			Category: cfg.JPush.HUAWEI.Category,
		},
		XIAOMI: struct {
			ChannelId string `json:"channel_id"`
		}{
			ChannelId: cfg.JPush.XIAOMI.ChannelId,
		},
		OPPO: struct {
			ChannelId string `json:"channel_id"`
		}{
			ChannelId: cfg.JPush.OPPO.ChannelId,
		},
	})
	return client
}
