package grade

type GetGradeByTermReq struct {
	Xnm int64 `form:"xnm,omitempty"` //学年名:例如2023表示2023~2024学年
	Xqm int64 `form:"xqm,omitempty"` //学期名:0表示所有学期,1表示第一学期,2表示第二学期,3表示第三学期
}

type GetGradeByTermResp struct {
	Grades []Grade // 课程信息
}

type Grade struct {
	Kcmc                string  `form:"kcmc,omitempty"`                //课程名称
	Xf                  float32 `form:"xf,omitempty"`                  //学分
	Cj                  float32 `form:"cj,omitempty"`                  //最终成绩
	Kcxzmc              string  `form:"kcxzmc,omitempty"`              //课程性质名称
	Kclbmc              string  `form:"Kclbmc,omitempty"`              //课程类别名称
	Kcbj                string  `form:"kcbj,omitempty"`                //课程标记(主修/辅修)
	Jd                  float32 `form:"jd,omitempty"`                  //绩点
	RegularGradePercent string  `form:"regularGradePercent,omitempty"` //平时成绩占比
	RegularGrade        float32 `form:"regularGrade,omitempty"`        //平时成绩分数
	FinalGradePercent   string  `form:"finalGradePercent,omitempty"`   ///期末成绩占比
	FinalGrade          float32 `form:"finalGrade,omitempty"`          //期末成绩分数
}

type GetGradeScoreResp struct {
	TypeOfGradeScores []TypeOfGradeScore `json:"type_of_grade_scores"`
}

type TypeOfGradeScore struct {
	Kcxzmc         string        `json:"kcxzmc,omitempty"` //课程性质名称
	GradeScoreList []*GradeScore `json:"grade_score_list,omitempty"`
}

type GradeScore struct {
	Kcmc string  `json:"kcmc,omitempty"` //课程名称
	Xf   float32 `json:"xf,omitempty"`   //学分
}
