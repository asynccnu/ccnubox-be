package domain

import (
	"time"
)

type Calendar struct {
	Year      int64
	Link      string
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Banner struct {
	WebLink     string
	PictureLink string
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Department struct {
	Name      string
	Phone     string
	Place     string
	Time      string
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

type InfoSum struct {
	Name        string
	Link        string
	Description string
	Image       string
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Website struct {
	Name        string
	Link        string
	Description string
	Image       string
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
