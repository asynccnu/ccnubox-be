package grade

type GetGradeByTermReq struct {
	Terms   []string `form:"terms"`   //学期筛选,格式为2024-1表示2024~2025学年第一学期
	Refresh bool     `form:"refresh"` //是否强制刷新,可选字段
	Kcxzmcs []string `form:"kcxzmcs"` //课程种类筛选,有如下类型:专业主干课程,通识选修课,通识必修课,个性发展课程,通识核心课等
}

type GetGradeByTermResp struct {
	Grades []Grade `json:"grades" binding:"required"` // 课程信息
}

type Grade struct {
	Xnm                 int64   `json:"xnm" binding:"required"`                 //学年
	Xqm                 int64   `json:"xqm" binding:"required"`                 //学期
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
