package dao

import "gorm.io/gorm"

type SeatDAO interface {
}

type seatDAO struct {
	db *gorm.DB
}

func NewSeatDAO(db *gorm.DB) SeatDAO {
	return &seatDAO{db: db}
}
