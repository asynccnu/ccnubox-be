package grade

type GetGradeByTermReq struct {
	Xnm     int64 `form:"xnm" binding:"required"` //学年名:例如2023表示2023~2024学年
	Xqm     int64 `form:"xqm" binding:"required"` //学期名:0表示所有学期,1表示第一学期,2表示第二学期,3表示第三学期
	Refresh bool  `form:"refresh"`                //是否强制刷新,可选字段
}

type GetGradeByTermResp struct {
	Grades []Grade `json:"grades" binding:"required"` // 课程信息
}

type Grade struct {
	Kcmc                string  `json:"kcmc" binding:"required"`                //课程名称
	Xf                  float32 `json:"xf" binding:"required"`                  //学分
	Cj                  float32 `json:"cj" binding:"required"`                  //最终成绩
	Kcxzmc              string  `json:"kcxzmc" binding:"required"`              //课程性质名称
	Kclbmc              string  `json:"Kclbmc" binding:"required"`              //课程类别名称
	Kcbj                string  `json:"kcbj" binding:"required"`                //课程标记(主修/辅修)
	Jd                  float32 `json:"jd" binding:"required"`                  //绩点
	RegularGradePercent string  `json:"regularGradePercent" binding:"required"` //平时成绩占比
	RegularGrade        float32 `json:"regularGrade" binding:"required"`        //平时成绩分数
	FinalGradePercent   string  `json:"finalGradePercent" binding:"required"`   ///期末成绩占比
	FinalGrade          float32 `json:"finalGrade" binding:"required"`          //期末成绩分数
}

type GetGradeScoreResp struct {
	TypeOfGradeScores []TypeOfGradeScore `json:"type_of_grade_scores" binding:"required"`
}

type TypeOfGradeScore struct {
	Kcxzmc         string        `json:"kcxzmc" binding:"required"` //课程性质名称
	GradeScoreList []*GradeScore `json:"grade_score_list" binding:"required"`
}

type GradeScore struct {
	Kcmc string  `json:"kcmc" binding:"required"` //课程名称
	Xf   float32 `json:"xf" binding:"required"`   //学分
}
