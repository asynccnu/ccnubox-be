package dao

import (
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(&model.Calendar{})
}
