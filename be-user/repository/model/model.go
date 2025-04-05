package model

type User struct {
	StudentId string `gorm:"column:student_id;unique;primaryKey;type:char(12)"`
	Password  string `gorm:"type:varchar(225)"`
	Utime     int64  `gorm:"autoUpdateTime"` // 更新时间，自动设置为当前时间戳（秒或毫秒）
	Ctime     int64  `gorm:"autoCreateTime"` // 创建时间，自动设置为当前时间戳（秒或毫秒）
}
