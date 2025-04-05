package model

import "gorm.io/gorm"

type InfoSum struct {
	Name        string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Link        string `gorm:"column:Link;type:VARCHAR(255)"`
	Description string `gorm:"column:Description;type:VARCHAR(255)"`
	Image       string `gorm:"column:Image;type:VARCHAR(255)"`
	gorm.Model
}
