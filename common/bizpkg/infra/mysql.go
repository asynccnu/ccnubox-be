package infra

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitMysql(cfg *conf.MysqlConf) *gorm.DB {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}
