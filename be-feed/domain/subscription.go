package domain

import "strings"

// 权限位顺序必须保持不变，兼容已存储的订阅位图。
const (
	EnergyPos = iota
	GradePos
	HolidayPos
	MuxiPos
	FeedBackPos
	LibraryPos
)

const (
	DefaultPushConfig uint16 = (1 << (LibraryPos + 1)) - 1
	// 配置缺失时普通类型沿用默认开启，图书馆必须显式创建配置后才能开启。
	MissingPushConfig uint16 = DefaultPushConfig &^ (1 << LibraryPos)
)

// 以权限位作为下标声明，避免调整顺序时与已存储的位图语义脱节。
var feedTypes = [...]string{
	EnergyPos:   "energy",
	GradePos:    "grade",
	HolidayPos:  "holiday",
	MuxiPos:     "muxi",
	FeedBackPos: "feedback",
	LibraryPos:  "library",
}

func NormalizeFeedType(label string) string {
	return strings.ToLower(label)
}

func FeedTypes() []string {
	return append([]string(nil), feedTypes[:]...)
}

func FeedTypePosition(label string) (int, bool) {
	label = NormalizeFeedType(label)
	for position, known := range feedTypes {
		if label == known {
			return position, true
		}
	}
	return 0, false
}

// PushConfigSnapshot 区分配置缺失、显式全关闭与软删除，读取时不创建配置。
type PushConfigSnapshot struct {
	Config  uint16
	Exists  bool
	Deleted bool
}

func (c PushConfigSnapshot) EffectiveConfig() uint16 {
	if c.Deleted {
		return 0
	}
	if !c.Exists {
		return MissingPushConfig
	}
	return c.Config
}

func (c PushConfigSnapshot) Allows(label string) bool {
	position, known := FeedTypePosition(label)
	return known && c.EffectiveConfig()&(1<<position) != 0
}
