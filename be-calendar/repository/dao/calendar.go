package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/model"
	"gorm.io/gorm"
)

// CalendarDAO 接口定义
type CalendarDAO interface {
	GetCalendar(ctx context.Context, year int64) (*model.Calendar, error)
	SaveCalendar(ctx context.Context, calendar *model.Calendar) error
	DelCalendar(ctx context.Context, year int64) (*model.Calendar, error)
}

// calendarDAO 结构体实现 CalendarDAO 接口
type calendarDAO struct {
	gorm *gorm.DB
}

// NewMysqlCalendarDAO 创建一个基于 MySQL 的 CalendarDAO 实现
func NewMysqlCalendarDAO(db *gorm.DB) CalendarDAO {
	return &calendarDAO{gorm: db}
}

// GetCalendars 获取日历数据
func (dao *calendarDAO) GetCalendar(ctx context.Context, year int64) (*model.Calendar, error) {
	var c model.Calendar
	err := dao.gorm.WithContext(ctx).Model(model.Calendar{}).Where("year=?", year).First(&c).Error
	return &c, err
}

// SaveCalendars 保存日历数据
func (dao *calendarDAO) SaveCalendar(ctx context.Context, calendar *model.Calendar) error {
	return dao.gorm.WithContext(ctx).Model(model.Calendar{}).Where("year=?", calendar.Year).Save(calendar).Error
}

// DelCalendar 删除日历数据
func (dao *calendarDAO) DelCalendar(ctx context.Context, year int64) (*model.Calendar, error) {
	var c model.Calendar
	err := dao.gorm.WithContext(ctx).Model(model.Calendar{}).Where("year=?", year).Delete(&c).Error
	return &c, err
}
