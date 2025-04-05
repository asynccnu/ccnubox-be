package dao

import (
	"github.com/asynccnu/ccnubox-be/be-elecprice/repository/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	err := db.AutoMigrate(&model.ElecpriceConfig{})
	if err != nil {
		return err
	}
	return nil
}
