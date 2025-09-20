package dao

import (
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	err := db.AutoMigrate(&model.Grade{}, &model.GraduateGrade{})
	if err != nil {
		return err
	}
	return nil
}
