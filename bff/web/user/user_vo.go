package user

type LoginByCCNUReq struct {
	StudentId string `json:"student_id" binding:"required"`
	Password  string `json:"password" binding:"required"` // 密码
}

type UserEditReq struct {
	Avatar     string `json:"avatar" binding:"required"`
	Nickname   string `json:"nickname" binding:"required"`
	UsingTitle string `json:"using_title" binding:"required"`
}

// UserProfileVo 自己的信息
type UserProfileVo struct {
	Id                   int64           `json:"id" binding:"required"`
	StudentId            string          `json:"studentId" binding:"required"`
	Avatar               string          `json:"avatar" binding:"required"`
	Nickname             string          `json:"nickname" binding:"required"`
	New                  bool            `json:"new" binding:"required"` // 是否为新用户，新用户尚未编辑过个人信息
	GradeSharingIsSigned bool            `json:"grade_sharing_is_signed" binding:"required"`
	UsingTitle           string          `json:"using_title" binding:"required"`
	TitleOwnership       map[string]bool `json:"title_ownership" binding:"required"`
	Utime                int64           `json:"utime" binding:"required"`
	Ctime                int64           `json:"ctime" binding:"required"`
}

// UserPublicProfileVo 别人的信息
type UserPublicProfileVo struct {
	Id       int64  `json:"id" binding:"required"`
	Avatar   string `json:"avatar" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
}
