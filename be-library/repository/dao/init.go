package dao

import "gorm.io/gorm"

func InitTables(db *gorm.DB) error {
	err := db.AutoMigrate(&Comment{})
	if err != nil {
		return err
	}

	return nil
}
