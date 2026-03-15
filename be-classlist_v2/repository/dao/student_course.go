package dao

import "gorm.io/gorm"

type StudentCourseDBRepo struct {
	Mysql *gorm.DB
}
