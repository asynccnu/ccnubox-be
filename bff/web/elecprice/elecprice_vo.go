package elecprice

type SetStandardRequest struct {
	RoomName string `json:"room_name,omitempty"`
	RoomId   string `json:"room_id,omitempty"`
	Limit    int64  `json:"limit,omitempty"`
}

type Price struct {
	RemainMoney       string `json:"remain_money,omitempty"`
	YesterdayUseValue string `json:"yesterday_use_value,omitempty"`
	YesterdayUseMoney string `json:"yesterday_use_money,omitempty"`
}

type GetArchitectureRequest struct {
	AreaName string `form:"area_name,omitempty" json:"area_name,omitempty"`
}

type Architecture struct {
	ArchitectureName string `json:"architecture_name,omitempty"`
	ArchitectureID   string `json:"architecture_id,omitempty"`
	BaseFloor        string `json:"base_floor,omitempty"`
	TopFloor         string `json:"top_floor,omitempty"`
}

type GetArchitectureResponse struct {
	ArchitectureList []*Architecture `json:"architecture_list,omitempty"`
}

type GetRoomInfoRequest struct {
	ArchitectureID string `json:"architecture_id,omitempty" form:"architecture_id,omitempty"`
	Floor          string `json:"floor,omitempty" form:"floor,omitempty"`
}

type Room struct {
	RoomID   string `json:"room_id,omitempty"`
	RoomName string `json:"room_name,omitempty"`
}

type GetRoomInfoResponse struct {
	RoomList []*Room `json:"room_list,omitempty"`
}

type GetPriceRequest struct {
	RoomId string `json:"room_id,omitempty" form:"room_id,omitempty"`
}

type GetPriceResponse struct {
	Price *Price `json:"price,omitempty"`
}

type GetStandardListRequest struct {
	StudentId string `json:"student_id,omitempty" form:"student_id,omitempty"`
}

type Standard struct {
	RoomName string `json:"room_name,omitempty"`
	RoomId   string `json:"room_id,omitempty"`
	Limit    int64  `json:"limit,omitempty"`
}

type StandardResp struct {
	RoomName string `json:"room_name,omitempty"`
	Limit    int64  `json:"limit,omitempty"`
}
type GetStandardListResponse struct {
	StandardList []*StandardResp `json:"standard_list,omitempty"`
}

type CancelStandardRequest struct {
	RoomId string `json:"room_id,omitempty"`
}
