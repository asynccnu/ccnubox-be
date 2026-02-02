package dao

import (
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Calendar{},
		&model.Banner{},
		&model.Department{},
		&model.InfoSum{},
		&model.Website{},
		&model.Version{},
	)
}
