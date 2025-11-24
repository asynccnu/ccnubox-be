package biz

import (
	"context"
)

var RoomIDs = []string{
	"100455820", // 主馆图书馆一楼-一楼综合学习室
	"100455822", // 主馆图书馆二楼-二楼借阅室（一）
	"100671994", // 主馆图书馆二楼-二楼借阅室（二）
	"100455824", // 主馆图书馆三楼-三楼借阅室（三）
	"100455826", // 主馆图书馆四楼-四楼自主学习中心
	"100455828", // 主馆图书馆五楼-五楼借阅室（四）
	"100746476", // 主馆图书馆五楼-五楼借阅室（五）
	"100746204", // 主馆图书馆六楼-六楼阅览室（一）
	"100455830", // 主馆图书馆六楼-六楼外文借阅室
	"100455832", // 主馆图书馆七楼-七楼阅览室（二）
	"100746480", // 主馆图书馆七楼-七楼阅览室（三）
	"100455834", // 主馆图书馆九楼-九楼阅览室
	"101699179", // 南湖分馆一楼-南湖分馆一楼开敞座位区
	"101699187", // 南湖分馆一楼-南湖分馆一楼中庭开敞座位区
	"101699189", // 南湖分馆二楼-南湖分馆二楼开敞座位区
	"101699191", // 南湖分馆二楼-南湖分馆二楼卡座区
}

type Seat struct {
	LabName  string // 南湖分馆一楼
	RoomID   string // room_id
	RoomName string // 南湖分馆一楼开敞座位区
	DevID    string // 101699849 或称 seatid
	DevName  string // N1245 或称 seatname
	Ts       []*TimeSlot
}

type TimeSlot struct {
	Start  string
	End    string
	State  string
	Owner  string
	Occupy bool
}

// SeatFilter 座位查询过滤器
type SeatFilter struct {
	RoomID    string
	TimeStart string
	TimeEnd   string
}

// SeatStatistics 座位统计信息
type SeatStatistics struct {
	Total     int64   `json:"total"`
	Available int64   `json:"available"`
	Partial   int64   `json:"partial"`
	Busy      int64   `json:"busy"`
	UsageRate float64 `json:"usageRate"`
}

type SeatRepo interface {
	// 核心方法：从爬虫同步数据（要修改，应该是通过 crawler 直接将座位同步到里面）
	// SyncSeatsIntoSQL(ctx context.Context, roomID string, stuID string, seats []*Seat) error

	// 查询方法
	// Get(ctx context.Context, devID string) (*Seat, error)
	// GetByRoom(ctx context.Context, roomID string) ([]*Seat, error)
	// GetAvailableSeats(ctx context.Context, filter *SeatFilter) ([]*Seat, int64, error)
	// GetStatistics(ctx context.Context, roomID string) (*SeatStatistics, error)
	FindFirstAvailableSeat(ctx context.Context, start, end int64, roomID []string) (string, bool, error)
	// 获取所有楼层座位信息
	GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*Seat, error)

	// 缓存方法
	SaveRoomSeatsInRedis(ctx context.Context, stuID string, roomID []string) error

	// 更新方法
	// UpdateTimeSlots(ctx context.Context, devID string, timeSlots []*TimeSlot) error
}
