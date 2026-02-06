package domain

import (
	"encoding/json"

	elecpricev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/elecprice/v1"
)

type ElectricMSG struct {
	RoomName  *string
	StudentId string // 学号
	Remain    *string
	Limit     *int64
	RoomID    *string
}

type ResultInfo struct {
	Result    string `xml:"result"`
	TimeStamp string `xml:"timeStamp"`
	Msg       string `xml:"msg"`
}

type Architecture struct {
	ArchitectureID     string `xml:"ArchitectureID" json:"architectureID"`
	ArchitectureName   string `xml:"ArchitectureName" json:"architectureName"`
	ArchitectureStorys string `xml:"ArchitectureStorys" json:"architectureStorys"`
	ArchitectureBegin  string `xml:"ArchitectureBegin" json:"architectureBegin"`
}

type ArchitectureInfoList struct {
	ArchitectureInfo []Architecture `xml:"architectureInfo" json:"architectureInfo"`
}

func (al *ArchitectureInfoList) Marshal() string {
	if bytes, err := json.Marshal(al); err == nil {
		return string(bytes)
	}
	return ""
}

func (al *ArchitectureInfoList) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), al)
}

type ResultArchitectureInfo struct {
	ResultInfo           ResultInfo           `xml:"resultInfo"`
	ArchitectureInfoList ArchitectureInfoList `xml:"architectureInfoList"`
}

type Prices struct {
	AC    PriceInfo
	Light PriceInfo
	Union PriceInfo
}

type PriceInfo struct {
	RemainMoney       string
	YesterdayUseValue string
	YesterdayUseMoney string
}

func (pi *PriceInfo) ToProto() *elecpricev1.Price {
	return &elecpricev1.Price{
		RemainMoney:       pi.RemainMoney,
		YesterdayUseValue: pi.YesterdayUseValue,
		YesterdayUseMoney: pi.YesterdayUseMoney,
	}
}

type Standard struct {
	Limit    int64
	RoomId   string
	RoomName string
}

type RoomInfo struct {
	RoomName string `json:"roomName"`
	AC       string `json:"ac"`
	Light    string `json:"light"`
	Union    string `json:"union"`
}

func (ri *RoomInfo) Marshal() string {
	if bytes, err := json.Marshal(ri); err == nil {
		return string(bytes)
	}
	return ""
}

func (ri *RoomInfo) IsUnion() bool {
	return ri.Union != ""
}

func (ri *RoomInfo) Unmarshal(data string) {
	_ = json.Unmarshal([]byte(data), ri)
}

type RoomInfoList struct {
	Rooms []RoomInfo `json:"rooms"`
}

func (rl *RoomInfoList) Marshal() string {
	if bytes, err := json.Marshal(rl); err == nil {
		return string(bytes)
	}
	return ""
}

func (rl *RoomInfoList) Unmarshal(data string) {
	// 这里在其他地方做了放空处理
	_ = json.Unmarshal([]byte(data), rl)
}

type SetStandardRequest struct {
	StudentId string
	Standard  *Standard
}

type SetStandardResponse struct {
}

type GetStandardListRequest struct {
	StudentId string
}

type GetStandardListResponse struct {
	Standard []*Standard
}

type CancelStandardRequest struct {
	StudentId string
	RoomId    string
}

type CancelStandardResponse struct{}
