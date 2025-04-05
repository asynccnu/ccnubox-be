package dao

import (
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {

	//创建用户配置表
	err := db.AutoMigrate(
		&model.FeedEvent{},
		&model.UserFeedConfig{},
		&model.Token{},
		&model.FeedFailEvent{},
	)
	if err != nil {
		return err
	}

	return nil
}
