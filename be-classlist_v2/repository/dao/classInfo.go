package dao

import (
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm"
)

type ClassInfoDBRepo struct {
	Mysql *gorm.DB
}

func NewClassInfoDBRepo(mysqlDB *gorm.DB, logger logger.Logger) (*ClassInfoDBRepo, func(), error) {
	cleanup := func() {
		logger.Info("closing mysql resources")
	}
	return &ClassInfoDBRepo{
		Mysql: mysqlDB,
	}, cleanup, nil
}
