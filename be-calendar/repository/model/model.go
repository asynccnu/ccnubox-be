package model

import "gorm.io/gorm"

type Calendar struct {
	Year int64  `gorm:"column:year;unique"`
	Link string `gorm:"column:link"`
	gorm.Model
}
