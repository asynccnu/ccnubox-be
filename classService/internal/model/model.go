package model

type ClassInfo struct {
	ID           string  `gorm:"primaryKey;column:id" json:"id"`                     //集合了课程信息的字符串，便于标识（课程ID）
	Day          int64   `gorm:"column:day;not null" json:"day"`                     //星期几
	Teacher      string  `gorm:"column:teacher;not null" json:"teacher"`             //任课教师
	Where        string  `gorm:"column:where;not null" json:"where"`                 //上课地点
	ClassWhen    string  `gorm:"column:class_when;not null" json:"class_when"`       //上课是第几节（如1-2,3-4）
	WeekDuration string  `gorm:"column:week_duration;not null" json:"week_duration"` //上课的周数
	Classname    string  `gorm:"column:class_name;not null" json:"classname"`        //课程名称
	Credit       float64 `gorm:"column:credit;default:1.0" json:"credit"`            //学分
	Weeks        int64   `gorm:"column:weeks;not null" json:"weeks"`                 //哪些周
	Semester     string  `gorm:"column:semester;not null" json:"semester"`           //学期
	Year         string  `gorm:"column:year;not null" json:"year"`                   //学年
}

type CTime struct {
	Weeks    []int //有哪些周
	Day      int   //在星期几
	Sections []int //有哪几节
}

type CTWPair struct {
	CT    CTime  // 上课时间
	Where string // 上课地点
}
