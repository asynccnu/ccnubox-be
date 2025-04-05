package class

type GetClassListRequest struct {
	Year     string `form:"year"`
	Semester string `form:"semester"`
	Refresh  bool   `form:"refresh"`
}

type ClassInfo struct {
	ID           string  `json:"id"`            //集合了课程信息的字符串，便于标识（课程ID）
	Day          int64   `json:"day"`           //星期几
	Teacher      string  `json:"teacher"`       //任课教师
	Where        string  `json:"where"`         //上课地点
	ClassWhen    string  `json:"class_when"`    //上课是第几节（如1-2,3-4）
	WeekDuration string  `json:"week_duration"` //上课的周数
	Classname    string  `json:"classname"`     //课程名称
	Credit       float64 `json:"credit"`        //学分
	Weeks        []int   `json:"weeks"`         //哪些周
	Semester     string  `json:"semester"`      //学期
	Year         string  `json:"year"`          //学年
}

type AddClassRequest struct {

	// 课程名称
	Name string `json:"name,omitempty"`
	// 第几节 '形如 "1-3","1-1"'
	DurClass string `json:"dur_class,omitempty"`
	// 地点
	Where string `json:"where,omitempty"`
	// 教师
	Teacher string `json:"teacher,omitempty"`
	// 哪些周
	Weeks []int `json:"weeks,omitempty"`
	// 学期
	Semester string `json:"semester,omitempty"`
	// 学年
	Year string `json:"year,omitempty"`
	// 星期几
	Day int64 `json:"day,omitempty"`
	// 学分
	Credit *float64 `json:"credit,omitempty"`
}
type DeleteClassRequest struct {
	// 要被删的课程id
	Id string `json:"id,omitempty"`

	// 学年  "2024" -> 代表"2024-2025学年"
	Year string `json:"year,omitempty"`
	// 学期 "1"代表第一学期，"2"代表第二学期，"3"代表第三学期
	Semester string `json:"semester,omitempty"`
}
type UpdateClassRequest struct {

	// 课程名称
	Name *string ` json:"name,omitempty"`
	// 第几节 '形如 "1-3","1-1"'
	DurClass *string ` json:"dur_class,omitempty"`
	// 地点
	Where *string ` json:"where,omitempty"`
	// 教师
	Teacher *string ` json:"teacher,omitempty"`
	// 哪些周
	Weeks []int ` json:"weeks,omitempty"`
	// 学期
	Semester string ` json:"semester,omitempty"`
	// 学年
	Year string ` json:"year,omitempty"`
	// 星期几
	Day *int64 ` json:"day,omitempty"`
	// 学分
	Credit *float64 ` json:"credit,omitempty"`
	// 课程的ID（唯一标识） 更新后这个可能会换，所以响应的时候会把新的ID返回
	ClassId string ` json:"classId,omitempty"`
}
type GetRecycleBinClassRequest struct {

	// 学年  "2024" 代表"2024-2025学年"
	Year string ` json:"year,omitempty"`
	// 学期 "1"代表第一学期，"2"代表第二学期，"3"代表第三学期
	Semester string ` json:"semester,omitempty"`
}
type RecoverClassRequest struct {
	// 学年  "2024" 代表"2024-2025学年"
	Year string ` json:"year,omitempty"`
	// 学期 "1"代表第一学期，"2"代表第二学期，"3"代表第三学期
	Semester string ` json:"semester,omitempty"`
	// 课程的ID（唯一标识） 更新后这个可能会换，所以响应的时候会把新的ID返回
	ClassId string ` json:"classId,omitempty"`
}

type SearchRequest struct {
	// 搜索关键词,匹配的是课程名称和教师姓名
	SearchKeyWords string `form:"searchKeyWords,omitempty"`
	Year           string `form:"year,omitempty"`
	Semester       string `form:"semester,omitempty"`
}

//type Class struct {
//	Info     []*ClassInfo `json:"info"`
//}

type GetClassListResp struct {
	Classes []*ClassInfo `json:"classes"`
}

type GetRecycleBinClassInfosReq struct {
	Year     string `form:"year"`
	Semester string `form:"semester"`
}
type SearchClassResp struct {
	ClassInfos []*ClassInfo `json:"classInfos"`
}

type GetRecycleBinClassInfosResp struct {
	ClassInfos []*ClassInfo `json:"classInfos"`
}
type GetSchoolDayReq struct{}

type GetSchoolDayResp struct {
	HolidayTime int64 `json:"holiday_time"`
	SchoolTime  int64 `json:"school_time"`
}
