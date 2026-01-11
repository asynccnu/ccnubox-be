package model

import "gorm.io/gorm"

type Content interface {
	Calendar | Banner | Department | InfoSum | Website
	Type() string
}

type Calendar struct {
	Year int64  `gorm:"column:year;unique"`
	Link string `gorm:"column:link"`
	gorm.Model
}

func (Calendar) Type() string {
	return "calendar"
}

type Banner struct {
	WebLink     string `gorm:"column:web_link;type:VARCHAR(255);not null"`
	PictureLink string `gorm:"column:picture_link;type:VARCHAR(255);not null"`
	gorm.Model
}

func (Banner) Type() string {
	return "banner"
}

type Department struct {
	Name  string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Phone string `gorm:"column:Phone;type:VARCHAR(50)"`
	Place string `gorm:"column:Place;type:VARCHAR(255)"`
	Time  string `gorm:"column:Time;type:VARCHAR(255)"`
	gorm.Model
}

func (Department) Type() string {
	return "department"
}

type InfoSum struct {
	Name        string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Link        string `gorm:"column:Link;type:VARCHAR(255)"`
	Description string `gorm:"column:Description;type:VARCHAR(255)"`
	Image       string `gorm:"column:Image;type:VARCHAR(255)"`
	gorm.Model
}

func (InfoSum) Type() string {
	return "infosum"
}

type Website struct {
	Name        string `gorm:"column:Name;type:VARCHAR(255);not null"`
	Link        string `gorm:"column:Link;type:VARCHAR(255)"`
	Description string `gorm:"column:Description;type:VARCHAR(255)"`
	Image       string `gorm:"column:Image;type:VARCHAR(255)"`
	gorm.Model
}

func (Website) Type() string {
	return "website"
}
