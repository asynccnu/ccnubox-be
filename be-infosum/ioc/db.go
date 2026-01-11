package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-infosum/conf"
	"github.com/asynccnu/ccnubox-be/be-infosum/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func InitDB(l logger.Logger, cfg *conf.InfraConf) *gorm.DB {
	db, err := gorm.Open(mysql.Open(cfg.Mysql.DSN), &gorm.Config{
		Logger: glogger.New(gormLoggerFunc(l.Debug), glogger.Config{
			SlowThreshold: 0,
			LogLevel:      glogger.Info, // 以Debug模式打印所有Info级别能产生的gorm日志
		}),
	})
	if err != nil {
		panic(err)
	}
	err = dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return db
}

type gormLoggerFunc func(msg string, fields ...logger.Field)

func (g gormLoggerFunc) Printf(s string, i ...interface{}) {
	g(s, logger.Field{Key: "args", Val: i})
}
