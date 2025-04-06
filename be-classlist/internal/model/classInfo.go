package model

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type ClassInfo struct {
	ID        string    `gorm:"type:varchar(255);primaryKey;column:id" json:"id"` //集合了课程信息的字符串，便于标识（课程ID）
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	//ClassId      string  `gorm:"column:class_id" json:"class_id"`           //课程编号
	JxbId        string  `gorm:"type:varchar(100);column:jxb_id" json:"jxb_id"`                        //教学班ID
	Day          int64   `gorm:"column:day;not null" json:"day"`                                       //星期几
	Teacher      string  `gorm:"type:varchar(255);column:teacher;not null" json:"teacher"`             //任课教师
	Where        string  `gorm:"type:varchar(255);column:where;not null" json:"where"`                 //上课地点
	ClassWhen    string  `gorm:"type:varchar(255);column:class_when;not null" json:"class_when"`       //上课是第几节（如1-2,3-4）
	WeekDuration string  `gorm:"type:varchar(255);column:week_duration;not null" json:"week_duration"` //上课的周数
	Classname    string  `gorm:"type:varchar(255);column:class_name;not null" json:"classname"`        //课程名称
	Credit       float64 `gorm:"column:credit;default:1.0" json:"credit"`                              //学分
	Weeks        int64   `gorm:"column:weeks;not null" json:"weeks"`                                   //哪些周
	Semester     string  `gorm:"type:varchar(1);column:semester;not null" json:"semester"`             //学期
	Year         string  `gorm:"type:varchar(5);column:year;not null" json:"year"`                     //学年
}

func (ci *ClassInfo) TableName() string {
	return ClassInfoTableName
}
func (ci *ClassInfo) BeforeUpdate(tx *gorm.DB) (err error) {
	ci.UpdatedAt = time.Now()
	return
}

//func (ci *ClassInfo) SearchWeek(week int64) bool {
//	return (ci.Weeks & (1 << (week - 1))) != 0
//}

func (ci *ClassInfo) UpdateID() {
	ci.ID = fmt.Sprintf("Class:%s:%s:%s:%d:%s:%s:%s:%d", ci.Classname, ci.Year, ci.Semester, ci.Day, ci.ClassWhen, ci.Teacher, ci.Where, ci.Weeks)
}

func (ci *ClassInfo) String() string {
	val, _ := json.Marshal(*ci)
	return string(val)
}
