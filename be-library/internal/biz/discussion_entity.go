package biz

import "context"

//type Discussion struct {
//	LabID    string
//	LabName  string
//	KindID   string
//	KindName string
//	DevID    string
//	DevName  string
//	TS       []*DiscussionTS
//}

type Discussion struct {
	RoomID      string
	Name        string
	VenueID     string
	RoomType    string
	Address     string
	DisableList []*DisableTime
}

type DisableTime struct {
	Start string
	End   string
}

//type DiscussionTS struct {
//	Start  string
//	End    string
//	State  string
//	Title  string
//	Owner  string
//	Occupy bool
//}

type Search struct {
	ID    string `json:"id"`    // 预约研讨间id
	Pid   string `json:"Pid"`   // 学号
	Name  string `json:"name"`  // 姓名
	Label string `json:"label"` // 姓名(学号)
}

type DiscussionRepo interface {
	GetDiscussionInfos(ctx context.Context, stuID string) ([]*Discussion, error)
	SearchUserInfos(ctx context.Context, stuID string) (*Search, error)
}
