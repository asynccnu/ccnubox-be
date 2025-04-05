package domain

type Elecprice struct {
	Airconditioner *Prices `json:"airconditioner"`
	Lighting       *Prices `json:"lighting"`
}

type ElectricMSG struct {
	RoomName  *string
	StudentId string // 学号
	Remain    *string
}

type ResultInfo struct {
	Result    string `xml:"result"`
	TimeStamp string `xml:"timeStamp"`
	Msg       string `xml:"msg"`
}

type Architecture struct {
	ArchitectureID     string `xml:"ArchitectureID"`
	ArchitectureName   string `xml:"ArchitectureName"`
	ArchitectureStorys string `xml:"ArchitectureStorys"`
	ArchitectureBegin  string `xml:"ArchitectureBegin"`
}

type ArchitectureInfoList struct {
	ArchitectureInfo []Architecture `xml:"architectureInfo"`
}

type ResultArchitectureInfo struct {
	ResultInfo           ResultInfo           `xml:"resultInfo"`
	ArchitectureInfoList ArchitectureInfoList `xml:"architectureInfoList"`
}

type RoomInfo struct {
	RID  string
	Name string
}

type Prices struct {
	RemainMoney       string
	YesterdayUseValue string
	YesterdayUseMoney string
}

type Standard struct {
	Limit    int64
	RoomId   string
	RoomName string
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
