package model

import (
	"encoding/json"
)

const (
	ClassInfoTableName     string = "class_info"
	StudentCourseTableName string = "student_course"
	JxbTableName           string = "jxb"
)

type Class struct {
	Info *ClassInfo //课程信息
	//ThisWeek bool       //是否是本周
}

func (c *Class) String() string {
	val, _ := json.Marshal(*c)
	return string(val)
}

// Jxb 用来存取教学班
type Jxb struct {
	JxbId string `gorm:"type:varchar(100);column:jxb_id;uniqueIndex:idx_jxb,priority:1" json:"jxb_id"` // 教学班ID
	StuId string `gorm:"type:varchar(20);column:stu_id;uniqueIndex:idx_jxb,priority:2" json:"stu_id"`  // 学号
}

func (j *Jxb) TableName() string {
	return JxbTableName
}
