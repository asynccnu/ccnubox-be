package jpush

import (
	"github.com/Scorpio69t/jpush-api-golang-client"
	"github.com/mitchellh/mapstructure"
)

type client struct {
	pf          *jpush.Platform
	jPushClient *jpush.JPushClient
	o           *jpush.Options
}

type PushClient interface {
	Push(ids []string, pushData PushData) error
}

type PushData struct {
	ContentType string            `json:"content_type"`
	Extras      map[string]string `json:"extras"`
	MsgContent  string            `json:"msg_content"`
	Title       string            `json:"title"`
}

type JPushConfig struct {
	AppKey       string `json:"app_key"`
	MasterSecret string `json:"master_secret"`
	HUAWEI       struct {
		Category string `json:"category"`
	}
	XIAOMI struct {
		ChannelId string `json:"channel_id"`
	}
	OPPO struct {
		ChannelId string `json:"channel_id"`
	}
}

func NewJPushClient(cfg *JPushConfig) PushClient {
	//极光推送客户端
	var pf jpush.Platform
	//设定为推送给所有平台
	pf.All()
	//配置极光推送选项
	var o jpush.Options
	o.SetApnsProduction(true)
	o.AddThirdPartyChannel(jpush.XIAOMI, jpush.ThirdPartyOptions{ChannelId: cfg.XIAOMI.ChannelId})
	o.AddThirdPartyChannel(jpush.HUAWEI, jpush.ThirdPartyOptions{Category: cfg.HUAWEI.Category})
	o.AddThirdPartyChannel(jpush.OPPO, jpush.ThirdPartyOptions{ChannelId: cfg.OPPO.ChannelId})

	// 初始化极光推送客户端
	jPushClient := jpush.NewJPushClient(cfg.AppKey, cfg.MasterSecret)

	return &client{pf: &pf, o: &o, jPushClient: jPushClient}
}

func (c *client) Push(ids []string, pushData PushData) error {
	// 如果无推送目标直接跳过
	if len(ids) == 0 {
		return nil
	}

	//设置推送对象
	var at jpush.Audience

	at.SetID(ids)

	// 设置智能推送以及智能推送的内容
	var n jpush.Notification

	var extras map[string]interface{}
	extras = make(map[string]interface{}, len(pushData.Extras))
	for k, v := range pushData.Extras {
		extras[k] = v
	}

	//推送给所有的平台,包括安卓,ios,windows
	n.SetAndroid(&jpush.AndroidNotification{
		Alert:       pushData.MsgContent,
		AlertType:   7,
		BadgeAddNum: 1, //每次提醒增加的角标数量
		BuilderID:   1,
		Style:       0, //样式字段
		Title:       pushData.Title,
		Priority:    1,
		Extras:      extras,
	})

	n.SetIos(&jpush.IosNotification{
		Alert:             pushData.MsgContent,
		Badge:             1,
		ContentAvailable:  false,
		InterruptionLevel: "active",
		MutableContent:    true,
	})

	//加载推送
	payload := jpush.NewPayLoad()
	payload.SetOptions(c.o)
	payload.SetPlatform(c.pf)
	payload.SetAudience(&at)
	payload.SetNotification(&n)
	var interfaceMap map[string]interface{}

	// 使用 解码成map[string]interface{}
	err := mapstructure.Decode(pushData.Extras, &interfaceMap)
	if err != nil {
		return err
	}

	//将发送的消息改成byte类型
	data, err := payload.Bytes()
	if err != nil {
		return err
	}

	//发送消息推送
	_, err = c.jPushClient.Push(data)
	if err != nil {

		return err
	}
	return nil
}
