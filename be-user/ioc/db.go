package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-user/conf"
	"github.com/asynccnu/ccnubox-be/be-user/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/infra"
	"gorm.io/gorm"
)

func InitDB(cfg *conf.InfraConf) *gorm.DB {
	db := infra.InitMysql(cfg.Mysql)
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return db
}
