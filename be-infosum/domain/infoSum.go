package domain

import "gorm.io/gorm"

type InfoSum struct {
	Name        string
	Link        string
	Description string
	Image       string
	gorm.Model
}
