package elecprice

import elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"

type SetStandardRequest struct {
	RoomName string `json:"room_name" binding:"required"`
	RoomId   string `json:"room_id" binding:"required"`
	Limit    int64  `json:"limit" binding:"required"`
}

type Price struct {
	RemainMoney       string `json:"remain_money,omitempty"`
	YesterdayUseValue string `json:"yesterday_use_value,omitempty"`
	YesterdayUseMoney string `json:"yesterday_use_money,omitempty"`
}

func priceToVo(p *elecpricev1.GetPriceResponse_Price) Price {
	return Price{
		RemainMoney:       p.RemainMoney,
		YesterdayUseValue: p.YesterdayUseValue,
		YesterdayUseMoney: p.YesterdayUseMoney,
	}
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
	RoomName string `json:"room_name" binding:"required"`
	AC       string `json:"ac,omitempty"`
	Light    string `json:"light,omitempty"`
	Union    string `json:"union,omitempty"`
}

type GetRoomInfoResponse struct {
	RoomList []*Room `json:"room_list" binding:"required"`
}

type GetPriceRequest struct {
	RoomName string `json:"room_name" form:"room_name" binding:"required"`
}

type GetPriceResponse struct {
	AC    Price `json:"ac_price,omitempty"`
	Light Price `json:"light_price,omitempty"`
	Union Price `json:"union_price,omitempty"`
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
