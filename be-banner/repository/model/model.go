package model

import "gorm.io/gorm"

type Banner struct {
	WebLink     string `gorm:"column:web_link;type:VARCHAR(255);not null"`
	PictureLink string `gorm:"column:picture_link;type:VARCHAR(255);not null"`
	gorm.Model
}
