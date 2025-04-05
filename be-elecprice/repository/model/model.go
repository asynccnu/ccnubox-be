package model

import (
	"gorm.io/gorm"
	"time"
)

type ElecpriceConfig struct {
	StudentID string // 学生号
	Limit     int64  //金额
	TargetID  string // 房间ID
	RoomName  string // 房间名称
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
