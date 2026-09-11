package infra

import (
	"time"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	defaultMysqlMaxOpenConns           = 30
	defaultMysqlMaxIdleConns           = 10
	defaultMysqlConnMaxLifetimeSeconds = 30 * 60
	defaultMysqlConnMaxIdleTimeSeconds = 5 * 60
)

type mysqlPoolSettings struct {
	maxOpenConns int
	maxIdleConns int
	maxLifetime  time.Duration
	maxIdleTime  time.Duration
}

func InitMysql(cfg *conf.MysqlConf) *gorm.DB {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	settings := normalizeMysqlPoolSettings(cfg)
	sqlDB.SetMaxOpenConns(settings.maxOpenConns)
	sqlDB.SetMaxIdleConns(settings.maxIdleConns)
	sqlDB.SetConnMaxLifetime(settings.maxLifetime)
	sqlDB.SetConnMaxIdleTime(settings.maxIdleTime)

	return db
}

func normalizeMysqlPoolSettings(cfg *conf.MysqlConf) mysqlPoolSettings {
	settings := mysqlPoolSettings{
		maxOpenConns: defaultMysqlMaxOpenConns,
		maxIdleConns: defaultMysqlMaxIdleConns,
		maxLifetime:  time.Duration(defaultMysqlConnMaxLifetimeSeconds) * time.Second,
		maxIdleTime:  time.Duration(defaultMysqlConnMaxIdleTimeSeconds) * time.Second,
	}
	if cfg == nil {
		return settings
	}
	if cfg.MaxOpenConns > 0 {
		settings.maxOpenConns = cfg.MaxOpenConns
	}
	if cfg.MaxIdleConns > 0 {
		settings.maxIdleConns = cfg.MaxIdleConns
	}
	if settings.maxIdleConns > settings.maxOpenConns {
		settings.maxIdleConns = settings.maxOpenConns
	}
	if cfg.ConnMaxLifetimeSeconds > 0 {
		settings.maxLifetime = time.Duration(cfg.ConnMaxLifetimeSeconds) * time.Second
	}
	if cfg.ConnMaxIdleTimeSeconds > 0 {
		settings.maxIdleTime = time.Duration(cfg.ConnMaxIdleTimeSeconds) * time.Second
	}
	return settings
}
