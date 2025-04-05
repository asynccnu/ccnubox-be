package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"time"
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
	BaseModel
	Read         bool         `gorm:"column:read;type:BOOLEAN;not null"`
	Type         string       `gorm:"column:type;type:VARCHAR(255);not null"`
	StudentId    string       `gorm:"column:student_id;type:varchar(255);not null"` // 学生 ID，唯一
	Title        string       `gorm:"column:title;type:TEXT;not null"`              // 标题
	Content      string       `gorm:"column:content;type:TEXT"`                     // 内容
	ExtendFields ExtendFields `gorm:"column:extend_fields;type:TEXT"`               // 拓展字段
}

type FeedFailEvent struct {
	BaseModel
	Type         string       `gorm:"column:type;type:VARCHAR(255);not null"`
	StudentId    string       `gorm:"column:student_id;type:varchar(255);not null"` // 学生 ID
	Title        string       `gorm:"column:title;type:TEXT;not null"`              // 标题
	Content      string       `gorm:"column:content;type:TEXT"`                     // 内容
	ExtendFields ExtendFields `gorm:"column:extend_fields;type:TEXT"`               // 拓展字段
}

// 定义权限开关的关键位
const (
	EnergyPos = iota
	GradePos
	HolidayPos
	MuxiPos
)

// UserFeedConfig 表示用户的 Feed 配置
type UserFeedConfig struct {
	StudentId  string `gorm:"column:student_id;type:varchar(255);not null;uniqueIndex"`
	PushConfig uint16 `gorm:"column:push_config;type:SMALLINT UNSIGNED;not null;default:31"` // 16位二进制，默认值 0000 0000 0001 1111 (十进制 31)
	BaseModel
}

// Token 表，存储每个用户的推送 Token
type Token struct {
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
