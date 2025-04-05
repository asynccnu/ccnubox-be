package feedback_help

import "time"

type GetQuestionsResp struct {
	Questions []FrequentlyAskedQuestion `json:"questions"`
}

type CreateQuestionReq struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type ChangeQuestionReq struct {
	QuestionId int64  `json:"question_id"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
}

type DeleteQuestionReq struct {
	QuestionId int64 `json:"question_id"`
}

type FindQuestionsByNameReq struct {
	Question string `from:"question"`
}

type FindQuestionsByNameResp struct {
	Questions []FrequentlyAskedQuestion `json:"questions"`
}

type NoteQuestionReq struct {
	QuestionId int64 `json:"question_id"`
	IfOver     bool  `json:"if_over"`
}

type NoteMoreFeedbackSearchSkipReq struct {
	QuestionId int64 `json:"question_id"`
}

type NoteEventTrackingReq struct {
	Event int8 `json:"event"`
}

type FrequentlyAskedQuestion struct {
	Id       int64
	Question string `json:"question"`
	Answer   string `json:"answer"`
	//Utime      time.Time
	//Ctime      time.Time
	ClickTimes int64 //记录该问题点击次数  //More_feedback_Q&A
}

type EventTracking struct {
	Id    int64
	Ctime time.Time
	Event int8 `json:"event"`
}

// More_feedback_search_skip
type EventQuestion struct {
	Id         int64
	Ctime      time.Time
	QuestionId int64
}

// 问题解决情况
type Question struct {
	Id         int64 `gorm:"primaryKey,autoIncrement"`
	QuestionId int64 `json:"question_id"`
	IfOver     bool  `json:"if_over"`
	Ctime      time.Time
}
