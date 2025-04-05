package feed

type GetFeedEventsResp struct {
	FeedEvents []FeedEventVO `json:"feed_events"`
}

type FeedEvent struct {
	Id           int64             `json:"id"`
	Title        string            `json:"title"`
	Type         string            `json:"type"`
	Content      string            `json:"content"`
	CreatedAt    int64             `json:"created_at"` //Unix时间戳
	ExtendFields map[string]string `json:"extend_fields"`
}
type FeedEventVO struct {
	Id           int64             `json:"id"`
	Title        string            `json:"title"`
	Type         string            `json:"type"`
	Content      string            `json:"content"`
	CreatedAt    int64             `json:"created_at"` //Unix时间戳
	ExtendFields map[string]string `json:"extend_fields"`
	Read         bool              `json:"read"`
}
type MuxiOfficialMSG struct {
	Title        string            `json:"title"`
	Content      string            `json:"content"`
	ExtendFields map[string]string `json:"extend_fields"` //自定义拓展字段
	PublicTime   int64             `json:"public_time"`   //发布的时间
	Id           string            `json:"id"`
}

type ClearFeedEventReq struct {
	FeedId int64  `json:"feed_id,omitempty"` //如果feedid和status都被填写了,那么就会清除当前的feedid代表的feed消息且状态为设置的status的
	Status string `json:"status,omitempty"`  //有三个可选字段all表示清除所有消息,read表示清除所有已读消息,unread表示清除所有未读消息
}

type ReadFeedEventReq struct {
	FeedId int64 `json:"feed_id"`
}

type ChangeFeedAllowListReq struct {
	Grade   bool `json:"grade"`
	Muxi    bool `json:"muxi"`
	Holiday bool `json:"holiday"`
	Energy  bool `json:"energy"`
}

type GetFeedAllowListResp struct {
	Grade   bool `json:"grade"`
	Muxi    bool `json:"muxi"`
	Holiday bool `json:"holiday"`
	Energy  bool `json:"energy"`
}
type ChangeElectricityStandardReq struct {
	ElectricityStandard bool `json:"electricity_standard"`
}

type SaveFeedTokenReq struct {
	Token string `json:"token"`
}
type RemoveFeedTokenReq struct {
	Token string `json:"token"`
}

type PublicMuxiOfficialMSGReq struct {
	Title        string            `json:"title"`
	Content      string            `json:"content"`
	ExtendFields map[string]string `json:"extend_fields,omitempty"`
	LaterTime    int64             `json:"later_time"` //延迟多久发布(单位是秒)
}

type PublicMuxiOfficialMSGResp struct {
	Title        string            `json:"title"`
	Content      string            `json:"content"`
	PublicTime   string            `json:"public_time"`
	ExtendFields map[string]string `json:"extend_fields,omitempty"`
	Id           string            `json:"id"`
}

type StopMuxiOfficialMSGReq struct {
	Id string `json:"id"`
}

type GetToBePublicMuxiOfficialMSGResp struct {
	MSGList []MuxiOfficialMSG `json:"msg_list"`
}
