package domain

import "gorm.io/gorm"

type Department struct {
	Name  string
	Phone string
	Place string
	Time  string
	gorm.Model
}
