package errcode

type Err struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Err) Error() string {
	return e.Msg
}
func New(code int, msg string) *Err {
	return &Err{Code: code, Msg: msg}
}

var (
	Err_EsAddClassInfo      = New(450, "创建classInfo失败")
	Err_EsSearchClassInfo   = New(451, "查询classInfo失败")
	Err_FreeClassroomSearch = New(452, "查询freeClassroom失败")
	ErrCCNULogin            = New(453, "CCNU登录失败")
)
