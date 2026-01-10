package model

import "gorm.io/gorm"

type Calendar struct {
	Year int64  `gorm:"column:year;unique"`
	Link string `gorm:"column:link"`
	gorm.Model
}

type Banner struct {
	WebLink     string `gorm:"column:web_link;type:VARCHAR(255);not null"`
	PictureLink string `gorm:"column:picture_link;type:VARCHAR(255);not null"`
	gorm.Model
}

type Department struct {
	Name  string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Phone string `gorm:"column:Phone;type:VARCHAR(50)"`
	Place string `gorm:"column:Place;type:VARCHAR(255)"`
	Time  string `gorm:"column:Time;type:VARCHAR(255)"`
	gorm.Model
}

type InfoSum struct {
	Name        string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Link        string `gorm:"column:Link;type:VARCHAR(255)"`
	Description string `gorm:"column:Description;type:VARCHAR(255)"`
	Image       string `gorm:"column:Image;type:VARCHAR(255)"`
	gorm.Model
}

type Website struct {
	Name        string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Link        string `gorm:"column:Link;type:VARCHAR(255)"`
	Description string `gorm:"column:Description;type:VARCHAR(255)"`
	Image       string `gorm:"column:Image;type:VARCHAR(255)"`
	gorm.Model
}
