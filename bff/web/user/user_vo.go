package user

type LoginByCCNUReq struct {
	StudentId string `json:"student_id"`
	Password  string `json:"password"` // 密码
}

type UserEditReq struct {
	Avatar     string `json:"avatar"`
	Nickname   string `json:"nickname"`
	UsingTitle string `json:"using_title"`
}

// UserProfileVo 自己的信息
type UserProfileVo struct {
	Id                   int64           `json:"id"`
	StudentId            string          `json:"studentId"`
	Avatar               string          `json:"avatar"`
	Nickname             string          `json:"nickname"`
	New                  bool            `json:"new"` // 是否为新用户，新用户尚未编辑过个人信息
	GradeSharingIsSigned bool            `json:"grade_sharing_is_signed"`
	UsingTitle           string          `json:"using_title"`
	TitleOwnership       map[string]bool `json:"title_ownership"`
	Utime                int64           `json:"utime"`
	Ctime                int64           `json:"ctime"`
}

// UserPublicProfileVo 别人的信息
type UserPublicProfileVo struct {
	Id       int64  `json:"id"`
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}
