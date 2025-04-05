package domain

type Grade struct {
	JxbId               string  `json:"jxb_id"`                        //教学班id
	Kcmc                string  `json:"kcmc,omitempty"`                //课程名
	Xf                  float32 `json:"xf,omitempty"`                  //学分
	Cj                  float32 `gorm:"column:cj"`                     // 总成绩
	Kcxzmc              string  `json:"kcxzmc,omitempty"`              //课程性质名称 比如专业主干课程/通识必修课
	Kclbmc              string  `json:"Kclbmc,omitempty"`              //课程类别名称，比如专业课/公共课
	Kcbj                string  `json:"kcbj,omitempty"`                //课程标记，比如主修/辅修
	Jd                  float32 `json:"jd,omitempty"`                  // 绩点
	RegularGradePercent string  `json:"regularGradePercent,omitempty"` //平时成绩占比
	RegularGrade        float32 `json:"regularGrade,omitempty"`        //平时成绩
	FinalGradePercent   string  `json:"finalGradePercent,omitempty"`   //期末成绩占比
	FinalGrade          float32 `json:"finalGrade,omitempty"`          //期末成绩
}

type TypeOfGradeScore struct {
	Kcxzmc         string       `json:"kcxzmc,omitempty"`
	GradeScoreList []GradeScore `json:"gradeScoreList,omitempty"`
}

type GradeScore struct {
	Kcmc string  `json:"kcmc,omitempty"`
	Xf   float32 `json:"xf,omitempty"`
}
