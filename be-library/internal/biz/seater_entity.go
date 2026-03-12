package biz

import (
	"context"
)

var RoomIDs = []string{
	"1995332665354928128", // 主馆图书馆一楼-一楼综合学习室
	"1995332883496484864", // 主馆图书馆二楼-二楼借阅室（一）
	"1995333003986255872", // 主馆图书馆二楼-二楼借阅室（二）
	"2004018551720448000", // 主馆图书馆二楼-二楼朗读记忆亭
	"1995333152401702912", // 主馆图书馆二楼-二楼自习走廊
	"1995333344618266624", // 主馆图书馆三楼-三楼借阅室（三）
	"2004019032295411712", // 主馆图书馆三楼-三楼朗读记忆亭
	"1995334401171832832", // 主馆图书馆五楼-五楼借阅室（四）
	"1995334528976470016", // 主馆图书馆五楼-五楼借阅室（五）
	"2004016577205702656", // 主馆图书馆五楼-五楼自习走廊
	"1995334920397307904", // 主馆图书馆六楼-六楼阅览室（一）
	"1995335580547203072", // 主馆图书馆六楼-六楼外文借阅室
	"1995336024782716928", // 主馆图书馆七楼-七楼阅览室（二）
	"1995336144576233472", // 主馆图书馆七楼-七楼阅览室（三）
	"2004017054660104192", // 主馆图书馆七楼-七楼自习走廊
	"1995336495886942208", // 主馆图书馆九楼-九楼阅览室
	"1995338268580167680", // 南湖分馆一楼-南湖分馆一楼开敞座位区
	"1995338594917990400", // 南湖分馆一楼-南湖分馆一楼中庭开敞座位区
	"1995338850321743872", // 南湖分馆二楼-南湖分馆二楼开敞座位区
	"1995339150692630528", // 南湖分馆二楼-南湖分馆二楼卡座区
}

var VenueList = []string{
	"1993950824018821120", //南湖分馆
}

var DiscussionRoomType = []string{
	"2027360488621404160",
}

type Seat struct {
	ID        string
	Label     string
	Name      string
	Status    string
	AfterFree bool
	FreeList  []*FreeTime
}

type FreeTime struct {
	Start string
	End   string
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
	FindFirstAvailableSeat(ctx context.Context, start, end int64, roomID []string, stuID string) (string, bool, error)
	// 获取所有楼层座位信息
	GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*Seat, error)

	// 更新方法
	// UpdateTimeSlots(ctx context.Context, devID string, timeSlots []*TimeSlot) error
}
