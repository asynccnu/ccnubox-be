package biz

import (
	"fmt"
	"time"
)

type ClassInfoBO struct {
	ID           string //集合了课程信息的字符串，便于标识（课程ID）
	CreatedAt    time.Time
	UpdatedAt    time.Time
	JxbId        string  //教学班ID
	Day          int64   //星期几
	Teacher      string  //任课教师
	Where        string  //上课地点
	ClassWhen    string  //上课是第几节（如1-2,3-4）
	WeekDuration string  //上课的周数
	Classname    string  //课程名称
	Credit       float64 //学分
	Weeks        int64   //哪些周
	Semester     string  //学期
	Year         string  //学年
	Nature       string  //课程性质
	MetaData     ClassMetaDataBO
}

type ClassMetaDataBO struct {
	IsOfficial bool   // 是否为官方课程
	Note       string //备注
}

func (ci *ClassInfoBO) UpdateID() {
	ci.ID = fmt.Sprintf("Class:%s:%s:%s:%d:%s:%s:%s:%d", ci.Classname, ci.Year, ci.Semester, ci.Day, ci.ClassWhen, ci.Teacher, ci.Where, ci.Weeks)
}

const (
	Pending = "pending"
	Ready   = "ready"
	Failed  = "failed"
)

type ClassRefreshLogBO struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	StuID     string    `json:"stu_id" gorm:"column:stu_id;index:idx_stu_year_semester_updatedat,priority:1"`
	Year      string    `json:"year" gorm:"column:year;index:idx_stu_year_semester_updatedat,priority:2"`
	Semester  string    `json:"semester" gorm:"column:semester;index:idx_stu_year_semester_updatedat,priority:3"`
	Status    string    `json:"status" gorm:"column:status"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;index:idx_stu_year_semester_updatedat,priority:4,sort:desc"`
}

func (c *ClassRefreshLogBO) IsPending() bool {
	return c.Status == Pending
}
func (c *ClassRefreshLogBO) IsReady() bool {
	return c.Status == Ready
}
func (c *ClassRefreshLogBO) IsFailed() bool {
	return c.Status == Failed
}

type StudentCourse struct {
	StuID           string //学号
	ClaID           string //课程ID
	Year            string //学年
	Semester        string //学期
	IsManuallyAdded bool   //是否为手动添加
	Note            string //课程备注
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
