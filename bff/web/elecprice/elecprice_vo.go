package elecprice

type SetStandardRequest struct {
	RoomName string `json:"room_name" binding:"required"`
	RoomId   string `json:"room_id" binding:"required"`
	Limit    int64  `json:"limit" binding:"required"`
}

type Price struct {
	RemainMoney       string `json:"remain_money" binding:"required"`
	YesterdayUseValue string `json:"yesterday_use_value" binding:"required"`
	YesterdayUseMoney string `json:"yesterday_use_money" binding:"required"`
}

type GetArchitectureRequest struct {
	AreaName string `form:"area_name" json:"area_name" binding:"required"`
}

type Architecture struct {
	ArchitectureName string `json:"architecture_name" binding:"required"`
	ArchitectureID   string `json:"architecture_id" binding:"required"`
	BaseFloor        string `json:"base_floor" binding:"required"`
	TopFloor         string `json:"top_floor" binding:"required"`
}

type GetArchitectureResponse struct {
	ArchitectureList []*Architecture `json:"architecture_list" binding:"required"`
}

type GetRoomInfoRequest struct {
	ArchitectureID string `json:"architecture_id" form:"architecture_id" binding:"required"`
	Floor          string `json:"floor" form:"floor" binding:"required"`
}

type Room struct {
	RoomID   string `json:"room_id" binding:"required"`
	RoomName string `json:"room_name" binding:"required"`
}

type GetRoomInfoResponse struct {
	RoomList []*Room `json:"room_list" binding:"required"`
}

type GetPriceRequest struct {
	RoomId string `json:"room_id" form:"room_id" binding:"required"`
}

type GetPriceResponse struct {
	Price *Price `json:"price" binding:"required"`
}

type GetStandardListRequest struct {
	//StudentId string `json:"student_id" form:"student_id" binding:"required"`
}

type Standard struct {
	RoomName string `json:"room_name" binding:"required"`
	RoomId   string `json:"room_id" binding:"required"`
	Limit    int64  `json:"limit" binding:"required"`
}

type StandardResp struct {
	RoomName string `json:"room_name" binding:"required"`
	Limit    int64  `json:"limit" binding:"required"`
}
type GetStandardListResponse struct {
	StandardList []*StandardResp `json:"standard_list" binding:"required"`
}

type CancelStandardRequest struct {
	RoomId string `json:"room_id" binding:"required"`
}
