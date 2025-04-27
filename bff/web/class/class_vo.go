package class

type GetClassListRequest struct {
	Year     string `form:"year" binding:"required"` //学年,格式为"2024"代表"2024-2025学年"`
	Semester string `form:"semester" binding:"required"`
	Refresh  bool   `form:"refresh" binding:"required"`
}

type ClassInfo struct {
	ID           string  `json:"id" binding:"required"`            //集合了课程信息的字符串，便于标识（课程ID）
	Day          int64   `json:"day" binding:"required"`           //星期几
	Teacher      string  `json:"teacher" binding:"required"`       //任课教师
	Where        string  `json:"where" binding:"required"`         //上课地点
	ClassWhen    string  `json:"class_when" binding:"required"`    //上课是第几节（如1-2,3-4）
	WeekDuration string  `json:"week_duration" binding:"required"` //上课的周数
	Classname    string  `json:"classname" binding:"required"`     //课程名称
	Credit       float64 `json:"credit" binding:"required"`        //学分
	Weeks        []int   `json:"weeks" binding:"required"`         //哪些周
	Semester     string  `json:"semester" binding:"required"`      //学期
	Year         string  `json:"year" binding:"required"`          //学年
}

type AddClassRequest struct {

	// 课程名称
	Name string `json:"name" binding:"required"`
	// 第几节 '形如 "1-3","1-1"'
	DurClass string `json:"dur_class" binding:"required"`
	// 地点
	Where string `json:"where" binding:"required"`
	// 教师
	Teacher string `json:"teacher" binding:"required"`
	// 哪些周
	Weeks []int `json:"weeks" binding:"required"`
	// 学期
	Semester string `json:"semester" binding:"required"`
	// 学年
	Year string `json:"year" binding:"required"`
	// 星期几
	Day int64 `json:"day" binding:"required"`
	// 学分
	Credit *float64 `json:"credit"`
}
type DeleteClassRequest struct {
	// 要被删的课程id
	Id string `json:"id" binding:"required"`

	// 学年  "2024" -> 代表"2024-2025学年"
	Year string `json:"year" binding:"required"`
	// 学期 "1"代表第一学期，"2"代表第二学期，"3"代表第三学期
	Semester string `json:"semester" binding:"required"`
}
type UpdateClassRequest struct {

	// 课程名称
	Name *string ` json:"name" `
	// 第几节 '形如 "1-3","1-1"'
	DurClass *string ` json:"dur_class"`
	// 地点
	Where *string ` json:"where"`
	// 教师
	Teacher *string ` json:"teacher"`
	// 哪些周
	Weeks []int ` json:"weeks"`
	// 学期
	Semester string ` json:"semester" binding:"required"`
	// 学年
	Year string ` json:"year" binding:"required"`
	// 星期几
	Day *int64 ` json:"day"`
	// 学分
	Credit *float64 ` json:"credit"`
	// 课程的ID（唯一标识） 更新后这个可能会换，所以响应的时候会把新的ID返回
	ClassId string ` json:"classId" binding:"required"`
}

type RecoverClassRequest struct {
	// 学年  "2024" 代表"2024-2025学年"
	Year string ` json:"year" binding:"required"`
	// 学期 "1"代表第一学期，"2"代表第二学期，"3"代表第三学期
	Semester string ` json:"semester" binding:"required"`
	// 课程的ID（唯一标识） 更新后这个可能会换，所以响应的时候会把新的ID返回
	ClassId string ` json:"classId" binding:"required"`
}

type SearchRequest struct {
	// 搜索关键词,匹配的是课程名称和教师姓名
	SearchKeyWords string `form:"searchKeyWords" binding:"required"`
	Year           string `form:"year" binding:"required"`      //学年,格式为"2024"代表"2024-2025学年"
	Semester       string `form:"semester" binding:"required"`  //学期,格式为"1"代表第一学期，"2"代表第二学期，"3"代表第三学期
	Page           int    `form:"page" binding:"required"`      //页码
	PageSize       int    `form:"page_size" binding:"required"` //每页大小
}

type GetClassListResp struct {
	Classes         []*ClassInfo `json:"classes" binding:"required"`
	LastRefreshTime int64        `json:"last_refresh_time" binding:"required"` //上次刷新时间的时间戳,上海时区
}

type GetRecycleBinClassInfosReq struct {
	Year     string `form:"year" binding:"required"`     //学年,格式为"2024"代表"2024-2025学年"
	Semester string `form:"semester" binding:"required"` //学期,格式为"1"代表第一学期，"2"代表第二学期，"3"代表第三学期
}
type SearchClassResp struct {
	ClassInfos []*ClassInfo `json:"classInfos" binding:"required"`
}

type GetRecycleBinClassInfosResp struct {
	ClassInfos []*ClassInfo `json:"classInfos" binding:"required"`
}
type GetSchoolDayReq struct{}

type GetSchoolDayResp struct {
	HolidayTime int64 `json:"holiday_time" binding:"required"`
	SchoolTime  int64 `json:"school_time" binding:"required"`
}
