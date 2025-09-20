package grade

type GetGradeByTermReq struct {
	Terms   []string `form:"terms" json:"terms"`     //学期筛选,格式为2024-1表示2024~2025学年第一学期
	Refresh bool     `form:"refresh" json:"refresh"` //是否强制刷新,可选字段
	Kcxzmcs []string `form:"kcxzmcs" json:"kcxzmcs"` //课程种类筛选,有如下类型:专业主干课程,通识选修课,通识必修课,个性发展课程,通识核心课,教师教育选修,教师教育必修,大学英语分级教育等
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

type UpdateGraduateGradesReq struct {
	Xnm  int64 `json:"xnm"`  // 学年，可选
	Xqm  int64 `json:"xqm"`  // 学期(1/2/3)，可选
	Cjzt int64 `json:"cjzt"` // 成绩状态(0-全部；不填也可)
}

type UpdateGraduateGradesResp struct {
	Grades []GraduateGrade `json:"grades"`
}

type GraduateGrade struct {
	StudentNum      string  `json:"studentNum"`
	JxbId           string  `json:"jxbId"`
	Status          string  `json:"status"`
	Year            string  `json:"year"`
	Term            int64   `json:"term"`
	StudentID       string  `json:"studentID"`
	Name            string  `json:"name"`
	StudentCategory string  `json:"studentCategory"`
	College         string  `json:"college"`
	Major           string  `json:"major"`
	Grade           int64   `json:"grade"`
	ClassCode       string  `json:"classCode"`
	ClassName       string  `json:"className"`
	ClassNature     string  `json:"classNature"`
	Credit          float32 `json:"credit"`
	Point           float32 `json:"point"`
	GradePoints     float32 `json:"gradePoints"`
	IsAvailable     string  `json:"isAvailable"`
	IsDegree        string  `json:"isDegree"`
	SetCollege      string  `json:"setCollege"`
	ClassMark       string  `json:"classMark"`
	ClassCategory   string  `json:"classCategory"`
	ClassID         string  `json:"classID"`
	Teacher         string  `json:"teacher"`
}
