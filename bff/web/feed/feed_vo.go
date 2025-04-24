package feed

type GetFeedEventsResp struct {
	FeedEvents []FeedEventVO `json:"feed_events"`
}

type FeedEvent struct {
	Id           int64             `json:"id" binding:"required"`
	Title        string            `json:"title" binding:"required"`
	Type         string            `json:"type" binding:"required"`
	Content      string            `json:"content" binding:"required"`
	CreatedAt    int64             `json:"created_at" binding:"required"` //Unix时间戳
	ExtendFields map[string]string `json:"extend_fields" binding:"required"`
}
type FeedEventVO struct {
	Id           int64             `json:"id" binding:"required"`
	Title        string            `json:"title" binding:"required"`
	Type         string            `json:"type" binding:"required"`
	Content      string            `json:"content" binding:"required"`
	CreatedAt    int64             `json:"created_at" binding:"required"` //Unix时间戳
	ExtendFields map[string]string `json:"extend_fields" binding:"required"`
	Read         bool              `json:"read" binding:"required"`
}
type MuxiOfficialMSG struct {
	Title        string            `json:"title" binding:"required"`
	Content      string            `json:"content" binding:"required"`
	ExtendFields map[string]string `json:"extend_fields" binding:"required"` //自定义拓展字段
	PublicTime   int64             `json:"public_time" binding:"required"`   //发布的时间
	Id           string            `json:"id" binding:"required"`
}

type ClearFeedEventReq struct {
	FeedId int64  `json:"feed_id" binding:"required"` //如果feedid和status都被填写了,那么就会清除当前的feedid代表的feed消息且状态为设置的status的
	Status string `json:"status" binding:"required"`  //有三个可选字段all表示清除所有消息,read表示清除所有已读消息,unread表示清除所有未读消息
}

type ReadFeedEventReq struct {
	FeedId int64 `json:"feed_id" binding:"required"`
}

type ChangeFeedAllowListReq struct {
	Grade   bool `json:"grade" binding:"required"`
	Muxi    bool `json:"muxi" binding:"required"`
	Holiday bool `json:"holiday" binding:"required"`
	Energy  bool `json:"energy" binding:"required"`
}

type GetFeedAllowListResp struct {
	Grade   bool `json:"grade" binding:"required"`
	Muxi    bool `json:"muxi" binding:"required"`
	Holiday bool `json:"holiday" binding:"required"`
	Energy  bool `json:"energy" binding:"required"`
}
type ChangeElectricityStandardReq struct {
	ElectricityStandard bool `json:"electricity_standard" binding:"required"`
}

type SaveFeedTokenReq struct {
	Token string `json:"token" binding:"required"`
}
type RemoveFeedTokenReq struct {
	Token string `json:"token" binding:"required"`
}

type PublicMuxiOfficialMSGReq struct {
	Title        string            `json:"title" binding:"required"`
	Content      string            `json:"content" binding:"required"`
	ExtendFields map[string]string `json:"extend_fields" binding:"required"`
	LaterTime    int64             `json:"later_time" binding:"required"` //延迟多久发布(单位是秒)
}

type PublicMuxiOfficialMSGResp struct {
	Title        string            `json:"title" binding:"required"`
	Content      string            `json:"content" binding:"required"`
	PublicTime   string            `json:"public_time" binding:"required"`
	ExtendFields map[string]string `json:"extend_fields" binding:"required"`
	Id           string            `json:"id" binding:"required"`
}

type StopMuxiOfficialMSGReq struct {
	Id string `json:"id" binding:"required"`
}

type GetToBePublicMuxiOfficialMSGResp struct {
	MSGList []MuxiOfficialMSG `json:"msg_list" binding:"required"`
}
