package model

import "gorm.io/gorm"

type Department struct {
	Name  string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Phone string `gorm:"column:Phone;type:VARCHAR(50)"`
	Place string `gorm:"column:Place;type:VARCHAR(255)"`
	Time  string `gorm:"column:Time;type:VARCHAR(255)"`
	gorm.Model
}
