package dao

import (
	"github.com/asynccnu/ccnubox-be/be-website/repository/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(&model.Website{})
}
