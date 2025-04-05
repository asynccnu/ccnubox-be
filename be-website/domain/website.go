package domain

import "gorm.io/gorm"

type Website struct {
	Name        string
	Link        string
	Description string
	Image       string
	gorm.Model
}
