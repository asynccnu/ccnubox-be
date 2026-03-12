package data

import (
	"gorm.io/gorm"
	"time"
)

const (
	ClassRefreshLogTableName string = "class_refresh_log"
)

type ClassRefreshLog struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	StuID     string    `json:"stu_id" gorm:"column:stu_id;index:idx_stu_year_semester_updatedat,priority:1"`
	Year      string    `json:"year" gorm:"column:year;index:idx_stu_year_semester_updatedat,priority:2"`
	Semester  string    `json:"semester" gorm:"column:semester;index:idx_stu_year_semester_updatedat,priority:3"`
	Status    string    `json:"status" gorm:"column:status"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;index:idx_stu_year_semester_updatedat,priority:4,sort:desc"`
}

func (c *ClassRefreshLog) TableName() string {
	return ClassRefreshLogTableName
}

func (c *ClassRefreshLog) BeforeCreate(tx *gorm.DB) (err error) {
	c.UpdatedAt = time.Now()
	return
}

func (c *ClassRefreshLog) BeforeUpdate(tx *gorm.DB) (err error) {
	c.UpdatedAt = time.Now()
	return
}
