package model

// Grade 定义与数据库映射的结构体
type Grade struct {
	Studentid           string  `gorm:"column:student_id;type:varchar(100);not null;primaryKey"` // 学生号
	JxbId               string  `gorm:"column:jxb_id;type:varchar(100);primaryKey"`              // 教学班ID
	Kcmc                string  `gorm:"column:kcmc;type:varchar(255)"`                           // 课程名
	Xnm                 int64   // 学年
	Xqm                 int64   // 学期名
	Xf                  float32 `gorm:"column:xf"`                                     // 学分
	Kcxzmc              string  `gorm:"column:kcxzmc;type:varchar(255)"`               // 课程性质名称，比如专业主干课程/通识必修课
	Kclbmc              string  `gorm:"column:kclbmc;type:varchar(255)"`               // 课程类别名称，比如专业课/公共课
	Kcbj                string  `gorm:"column:kcbj;type:varchar(50)"`                  // 课程标记，比如主修/辅修
	Jd                  float32 `gorm:"column:jd"`                                     // 绩点
	RegularGradePercent string  `gorm:"column:regular_grade_percent;type:varchar(10)"` // 平时成绩占比
	RegularGrade        float32 `gorm:"column:regular_grade"`                          // 平时成绩
	FinalGradePercent   string  `gorm:"column:final_grade_percent;type:varchar(10)"`   // 期末成绩占比
	FinalGrade          float32 `gorm:"column:final_grade"`                            // 期末成绩
	Cj                  float32 `gorm:"column:cj"`                                     // 总成绩
}
