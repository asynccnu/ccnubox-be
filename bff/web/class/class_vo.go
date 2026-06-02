package class

type GetClassListRequest struct {
	Year     string `form:"year" example:"2025"`                        // 学年,格式为"2025"代表"2025-2026学年"
	Semester string `form:"semester" example:"2"`                       // 学期,"1"第一学期,"2"第二学期,"3"第三学期
	Refresh  *bool  `form:"refresh" binding:"required" example:"false"` // 是否强制刷新课表
}

type ClassInfo struct {
	ID           string  `json:"id" binding:"required" example:"Class:测试课程:2025:2:1:1-2:测试老师:测试教室:3"` // 课程ID
	Day          int64   `json:"day" binding:"required" example:"1"`                                  // 星期几,1-7表示周一到周日
	Teacher      string  `json:"teacher" binding:"required" example:"测试老师"`                           // 任课教师
	Where        string  `json:"where" binding:"required" example:"测试教室"`                             // 上课地点
	ClassWhen    string  `json:"class_when" binding:"required" example:"1-2"`                         // 上课节次
	WeekDuration string  `json:"week_duration" binding:"required" example:"1-2周"`                     // 上课周数文本
	Classname    string  `json:"classname" binding:"required" example:"测试课程"`                         // 课程名称
	Credit       float64 `json:"credit" binding:"required" example:"1"`                               // 学分
	Weeks        []int   `json:"weeks" binding:"required" example:"1,2"`                              // 上课周次数组
	Semester     string  `json:"semester" binding:"required" example:"2"`                             // 学期
	Year         string  `json:"year" binding:"required" example:"2025"`                              // 学年
	Note         string  `json:"note" binding:"required" example:"课前预习"`                              // 备注
	IsOfficial   bool    `json:"is_official" binding:"required" example:"true"`                       // 是否为官方课程
	Nature       string  `json:"nature" binding:"required" example:"专业主干课程"`                          // 课程性质
}

type AddClassRequest struct {

	// 课程名称
	Name string `json:"name" binding:"required" example:"测试课程"`
	// 上课节次,形如 "1-2","9-10"
	DurClass string `json:"dur_class" binding:"required" example:"1-2"`
	// 地点
	Where string `json:"where" binding:"required" example:"测试教室"`
	// 教师
	Teacher string `json:"teacher" binding:"required" example:"测试老师"`
	// 上课周次数组
	Weeks []int `json:"weeks" binding:"required" example:"1,2"`
	// 学期
	Semester string `json:"semester" binding:"required" example:"2"`
	// 学年
	Year string `json:"year" binding:"required" example:"2025"`
	// 星期几
	Day int64 `json:"day" binding:"required" example:"1"`
	// 学分
	Credit *float64 `json:"credit" example:"1"`
}
type DeleteClassRequest struct {
	// 要被删的课程id
	Id string `json:"id" binding:"required" example:"Class:测试课程:2025:2:1:1-2:测试老师:测试教室:3"`

	// 学年  "2024" -> 代表"2024-2025学年"
	Year string `json:"year" binding:"required" example:"2025"`
	// 学期 "1"代表第一学期，"2"代表第二学期，"3"代表第三学期
	Semester string `json:"semester" binding:"required" example:"2"`
}
type UpdateClassRequest struct {

	// 课程名称
	Name *string `json:"name" example:"测试课程"`
	// 上课节次,形如 "1-2","9-10"
	DurClass *string `json:"dur_class" example:"1-2"`
	// 地点
	Where *string `json:"where" example:"测试教室"`
	// 教师
	Teacher *string `json:"teacher" example:"测试老师"`
	// 上课周次数组
	Weeks []int `json:"weeks" example:"1,2"`
	// 学期
	Semester string `json:"semester" binding:"required" example:"2"`
	// 学年
	Year string `json:"year" binding:"required" example:"2025"`
	// 星期几
	Day *int64 `json:"day" example:"1"`
	// 学分
	Credit *float64 `json:"credit" example:"1"`
	// 课程的ID（唯一标识） 更新后这个可能会换，所以响应的时候会把新的ID返回
	ClassId string `json:"classId" binding:"required" example:"Class:测试课程:2025:2:1:1-2:测试老师:测试教室:3"`
}

type GetClassListResp struct {
	Classes         []*ClassInfo `json:"classes" binding:"required"`
	LastRefreshTime int64        `json:"last_refresh_time" binding:"required" example:"1717248000"` // 上次刷新时间的时间戳,上海时区
}

type GetSchoolDayReq struct{}

type GetSchoolDayResp struct {
	HolidayTime int64 `json:"holiday_time" binding:"required" example:"1751644800"` // 放假日期零点时间戳,秒级
	SchoolTime  int64 `json:"school_time" binding:"required" example:"1739721600"`  // 开学日期零点时间戳,秒级
}

type UpdateClassNoteReq struct {
	Semester string `json:"semester" binding:"required" example:"2"`                                  // 学期
	Year     string `json:"year" binding:"required" example:"2025"`                                   // 学年
	ClassId  string `json:"classId" binding:"required" example:"Class:测试课程:2025:2:1:1-2:测试老师:测试教室:3"` // 课程ID
	Note     string `json:"note" binding:"required" example:"课前预习"`                                   // 备注
}

type DeleteClassNoteReq struct {
	Semester string `json:"semester" binding:"required" example:"2"`                                  // 学期
	Year     string `json:"year" binding:"required" example:"2025"`                                   // 学年
	ClassId  string `json:"classId" binding:"required" example:"Class:测试课程:2025:2:1:1-2:测试老师:测试教室:3"` // 课程ID
}
