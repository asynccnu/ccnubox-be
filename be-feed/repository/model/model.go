package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ExtendFields 是自定义类型，表示可以包含任意键值对的扩展字段,通过序列化和反序列化进行操作,实际使用量较小所以json也OK
type ExtendFields map[string]string

// 实现 gorm 的 Scanner 接口（从数据库加载数据）
func (t *ExtendFields) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	if err := json.Unmarshal(bytes, t); err != nil {
		return fmt.Errorf("failed to unmarshal ExtendFields: %w", err)
	}

	return nil
}

// 实现 gorm 的 Valuer 接口（将数据保存到数据库）
func (t ExtendFields) Value() (driver.Value, error) {
	bytes, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ExtendFields: %w", err)
	}
	return bytes, nil
}

// FeedEvent 表示 Feed 事件
type FeedEvent struct {
	ID           int64          `gorm:"primaryKey;autoIncrement;column:id;index:idx_feed_events_inbox,priority:4"`
	CreatedAt    int64          `gorm:"column:created_at;not null;index:idx_feed_events_inbox,priority:3"`
	UpdatedAt    int64          `gorm:"column:updated_at;not null"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index;index:idx_feed_events_inbox,priority:2"`
	Read         bool           `gorm:"column:read;type:BOOLEAN;not null"`
	Type         string         `gorm:"column:type;type:VARCHAR(255);not null"`
	StudentId    string         `gorm:"column:student_id;type:varchar(255);not null;index:idx_feed_events_inbox,priority:1;uniqueIndex:uidx_feed_events_recipient_dedupe,priority:1"` // 学生 ID
	Title        string         `gorm:"column:title;type:TEXT;not null"`                                                                                                              // 标题
	Content      string         `gorm:"column:content;type:TEXT"`                                                                                                                     // 内容
	Url          string         `gorm:"column:url;type:varchar(2047)"`                                                                                                                // 消息详情跳转路由
	ExtendFields ExtendFields   `gorm:"column:extend_fields;type:TEXT"`                                                                                                               // 拓展字段
	DedupeKey    string         `gorm:"column:dedupe_key;type:varchar(255);not null;uniqueIndex:uidx_feed_events_recipient_dedupe,priority:2"`
	Source       string         `gorm:"column:source;type:varchar(64)"`
	OccurredAt   int64          `gorm:"column:occurred_at;not null;default:0"`
}

type FeedFailEvent struct {
	ID           int64          `gorm:"primaryKey;autoIncrement;column:id"`
	CreatedAt    int64          `gorm:"column:created_at;not null"`
	UpdatedAt    int64          `gorm:"column:updated_at;not null"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index;index:idx_fail_stuid_deleted,priority:2"`
	Type         string         `gorm:"column:type;type:VARCHAR(255);not null"`
	StudentId    string         `gorm:"column:student_id;type:varchar(255);not null;index:idx_fail_stuid_deleted,priority:1"` // 学生 ID
	Title        string         `gorm:"column:title;type:TEXT;not null"`                                                      // 标题
	Content      string         `gorm:"column:content;type:TEXT"`                                                             // 内容
	Url          string         `gorm:"column:url;type:varchar(2047)"`                                                        // 消息详情跳转路由
	ExtendFields ExtendFields   `gorm:"column:extend_fields;type:TEXT"`                                                       // 拓展字段
}

func (f *FeedEvent) BeforeCreate(_ *gorm.DB) error {
	now := time.Now().Unix()
	f.CreatedAt = now
	f.UpdatedAt = now
	return nil
}

func (f *FeedEvent) BeforeUpdate(_ *gorm.DB) error {
	f.UpdatedAt = time.Now().Unix()
	return nil
}

func (f *FeedFailEvent) BeforeCreate(_ *gorm.DB) error {
	now := time.Now().Unix()
	f.CreatedAt = now
	f.UpdatedAt = now
	return nil
}

func (f *FeedFailEvent) BeforeUpdate(_ *gorm.DB) error {
	f.UpdatedAt = time.Now().Unix()
	return nil
}

// 定义权限开关的关键位
const (
	EnergyPos = iota
	GradePos
	HolidayPos
	MuxiPos
	FeedBackPos
	LibraryPos
)

const DefaultPushConfig uint16 = (1 << (LibraryPos + 1)) - 1

// FeedUserConfig 表示用户的 Feed 配置
type FeedUserConfig struct {
	StudentId       string `gorm:"column:student_id;type:varchar(255);not null;uniqueIndex"`
	PushConfig      uint16 `gorm:"column:push_config;type:SMALLINT UNSIGNED;not null"` // 默认开启由创建逻辑显式赋值，避免 GORM 将全关闭的 0 替换为默认值
	LibraryRevision int64  `gorm:"column:library_revision;not null;default:0"`
	BaseModel
}

type FeedUserConfigChange struct {
	Revision       int64  `gorm:"primaryKey;autoIncrement:false;column:revision;index:idx_feed_config_changes_student_revision,priority:2"`
	StudentId      string `gorm:"column:student_id;type:varchar(255);not null;index:idx_feed_config_changes_student_revision,priority:1"`
	LibraryEnabled bool   `gorm:"column:library_enabled;not null"`
	CreatedAt      int64  `gorm:"column:created_at;not null"`
}

const FeedUserConfigRevisionAllocatorID int64 = 1

// FeedUserConfigRevisionAllocator 串行分配偏好变更版本号。
type FeedUserConfigRevisionAllocator struct {
	ID       int64 `gorm:"primaryKey;autoIncrement:false;column:id"`
	Revision int64 `gorm:"column:revision;not null;default:0"`
}

func (c *FeedUserConfigChange) BeforeCreate(_ *gorm.DB) error {
	if c.CreatedAt == 0 {
		c.CreatedAt = time.Now().Unix()
	}
	return nil
}

const (
	PushDeliveryPending    = "pending"
	PushDeliverySending    = "sending"
	PushDeliverySent       = "sent"
	PushDeliveryFailed     = "failed"
	PushDeliverySuppressed = "suppressed"
)

// 可重试的消息投递
type FeedPushDelivery struct {
	BaseModel
	FeedEventID   int64  `gorm:"column:feed_event_id;not null;uniqueIndex"`
	StudentId     string `gorm:"column:student_id;type:varchar(255);not null;index"`
	CID           string `gorm:"column:cid;type:varchar(255);not null;default:''"`
	Status        string `gorm:"column:status;type:varchar(16);not null;default:pending;index:idx_feed_push_due,priority:1;index:idx_feed_push_priority_due,priority:1"`
	Priority      int    `gorm:"column:priority;not null;default:0;index:idx_feed_push_priority_due,priority:2,sort:desc"`
	Attempts      int    `gorm:"column:attempts;not null;default:0"`
	NextAttemptAt int64  `gorm:"column:next_attempt_at;not null;default:0;index:idx_feed_push_due,priority:2;index:idx_feed_push_priority_due,priority:3"`
	LastError     string `gorm:"column:last_error;type:text"`
}

// FeedUserToken 表，存储每个用户的推送 FeedUserToken
type FeedUserToken struct {
	StudentId string `gorm:"column:student_id;not null"`
	Token     string `gorm:"column:token;type:VARCHAR(255);not null"` // 单个 token
	BaseModel
}

// BaseModel 使用 Unix 时间戳替代 gorm.Model
type BaseModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement;column:id"` // 主键
	CreatedAt int64          `gorm:"column:created_at;not null"`         // 创建时间
	UpdatedAt int64          `gorm:"column:updated_at;not null"`         // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`            // 软删除时间
}

// 设置 `CreatedAt` 和 `UpdatedAt` 自动更新
func (b *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().Unix()
	b.CreatedAt = now
	b.UpdatedAt = now
	return nil
}

func (b *BaseModel) BeforeUpdate(tx *gorm.DB) (err error) {
	b.UpdatedAt = time.Now().Unix()
	return nil
}
