package domain

type Grade struct {
	Xnm                 int64   `json:"xnm"`                           //学年
	Xqm                 int64   `json:"xqm"`                           //学期
	JxbId               string  `json:"jxb_id"`                        //教学班id
	Kcmc                string  `json:"kcmc,omitempty"`                //课程名
	Xf                  float32 `json:"xf,omitempty"`                  //学分
	Cj                  float32 `gorm:"column:cj"`                     //总成绩
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

type GraduateGrade struct {
	StudentID       string  `json:"studentID"`
	JxbId           string  `json:"jxbId"`
	Status          string  `json:"status"`
	Year            string  `json:"year"`
	Term            int64   `json:"term"`
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

// rank 部分
type GetRankByTermReq struct {
	StudentId string `json:"studentId"`
	XnmBegin  int64  `json:"xnm_begin"`
	XqmBegin  int64  `json:"xqm_begin"`
	XnmEnd    int64  `json:"xnm_end"`
	XqmEnd    int64  `json:"xqm_end"`
	Refresh   bool   `json:"refresh"`
}

type GetRankByTermResp struct {
	Rank    string
	Score   string
	Include []string
}

type LoadRankReq struct {
	StudentId string `json:"studentId"`
}
