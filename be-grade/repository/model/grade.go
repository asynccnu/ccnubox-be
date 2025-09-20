package model

// Grade 定义与数据库映射的结构体
type Grade struct {
	Studentid           string  `gorm:"column:student_id;type:varchar(100);not null;primaryKey"` // 学生号
	JxbId               string  `gorm:"column:jxb_id;type:varchar(100);not null;primaryKey"`     // 教学班ID
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

type GraduateGrade struct {
	StudentID       string  `gorm:"column:student_id;type:varchar(100);not null;primaryKey"` // 学号
	JxbId           string  `gorm:"column:jxb_id;type:varchar(100);not null;primaryKey"`     // 教学班ID
	Status          string  `gorm:"column:status;type:varchar(255)"`                         // 成绩审核状态
	Year            string  `gorm:"column:year;type:varchar(255)"`                           // 学年(2024-2025)
	Term            int64   `gorm:"column:term"`                                             // 学期
	Name            string  `gorm:"column:name;type:varchar(255)"`                           // 姓名
	StudentCategory string  `gorm:"column:student_category;type:varchar(255)"`               // 学生类别
	College         string  `gorm:"column:college;type:varchar(255)"`                        // 学院
	Major           string  `gorm:"column:major;type:varchar(255)"`                          // 专业
	Grade           int64   `gorm:"column:grade"`                                            // 年级
	ClassCode       string  `gorm:"column:code;type:varchar(255)"`                           // 课程代码
	ClassName       string  `gorm:"column:class_name;type:varchar(255)"`                     // 课程名称
	ClassNature     string  `gorm:"column:class_nature;type:varchar(255)"`                   // 课程性质
	Credit          float32 `gorm:"column:credit"`                                           // 学分
	Point           float32 `gorm:"column:point"`                                            // 成绩
	GradePoints     float32 `gorm:"column:grade_points"`                                     // 绩点
	IsAvailable     string  `gorm:"column:is_available;type:varchar(255)"`                   // 成绩是否作废
	IsDegree        string  `gorm:"column:is_degree;type:varchar(255)"`                      // 是否学位课程
	SetCollege      string  `gorm:"column:set_college;type:varchar(255)"`                    // 开课学院
	ClassMark       string  `gorm:"column:class_mark;type:varchar(255)"`                     // 课程标记
	ClassCategory   string  `gorm:"column:class_category;type:varchar(255)"`                 // 课程类别
	ClassID         string  `gorm:"column:class_id;type:varchar(255)"`                       // 教学班
	Teacher         string  `gorm:"column:teacher;type:varchar(255)"`                        // 任课教师
}
