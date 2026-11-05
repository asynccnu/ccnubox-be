package model

import (
	"strconv"
	"strings"
)

// Grade 定义与数据库映射的结构体
type Grade struct {
	StudentId           string  `gorm:"column:student_id;type:varchar(100);not null;primaryKey"` // 学生号
	KcId                string  //课程的id,主要用于确定是不是一门课
	JxbId               string  `gorm:"column:jxb_id;type:varchar(100);not null;primaryKey"` // 教学班ID
	Kcmc                string  `gorm:"column:kcmc;type:varchar(255)"`                       // 课程名
	Xnm                 int64   // 学年
	Xqm                 int64   // 学期名
	Xf                  float32 `gorm:"column:xf"`                                                // 学分
	Kcxzmc              string  `gorm:"column:kcxzmc;type:varchar(255)"`                          // 课程性质名称，比如专业主干课程/通识必修课
	Kclbmc              string  `gorm:"column:kclbmc;type:varchar(255)"`                          // 课程类别名称，比如专业课/公共课
	Kcbj                string  `gorm:"column:kcbj;type:varchar(50)"`                             // 课程标记，比如主修/辅修
	Jd                  float32 `gorm:"column:jd"`                                                // 绩点
	RegularGradePercent string  `gorm:"column:regular_grade_percent;type:varchar(10)"`            // 平时成绩占比
	RegularGrade        float32 `gorm:"column:regular_grade"`                                     // 平时成绩
	FinalGradePercent   string  `gorm:"column:final_grade_percent;type:varchar(10)"`              // 期末成绩占比
	FinalGrade          float32 `gorm:"column:final_grade"`                                       // 期末成绩
	Cj                  float32 `gorm:"column:cj"`                                                // 总成绩
	NotificationID      *string `gorm:"type:varchar(32);uniqueIndex:uidx_grades_notification_id"` // 稳定身份，历史未激活时为 NULL
	ScoreVersion        int64   `gorm:"not null;default:0"`                                       // 总成绩业务版本，0 表示未激活
	ChangeVersion       int64   `gorm:"column:change_version;not null;default:1"`                 // 成绩变更版本
}

// LegacyJxbID 只用于兼容本地历史记录，不能据此认定通知身份。
func (g Grade) LegacyJxbID() string {
	return g.Kcmc + strconv.FormatInt(g.Xnm, 10) + strconv.FormatInt(g.Xqm, 10)
}

// LocalJxbID 保留旧查询使用的本地键；元数据不足时不构造伪键。
func (g Grade) LocalJxbID() string {
	if strings.TrimSpace(g.JxbId) != "" {
		return g.JxbId
	}
	if strings.TrimSpace(g.Kcmc) == "" || g.Xnm <= 0 || g.Xqm < 1 || g.Xqm > 3 {
		return ""
	}
	return g.LegacyJxbID()
}

func (g Grade) HasRealJxbID() bool {
	return strings.TrimSpace(g.JxbId) != "" && g.JxbId != g.LegacyJxbID()
}

func (g Grade) NotificationIdentity() string {
	if g.NotificationID == nil {
		return ""
	}
	return *g.NotificationID
}

func (g Grade) ValidNotificationState() bool {
	if g.NotificationID == nil {
		return g.ScoreVersion == 0
	}
	return ValidNotificationID(*g.NotificationID) && g.ScoreVersion >= 1
}
