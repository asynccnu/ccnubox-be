package data

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewDB, NewRedisDB, NewAssembler, NewSeatRepo, NewCommentRepo, NewRecordRepo, NewCreditPointsRepo)

// Data 做CURD时使用该框架
type Data struct {
	db    *gorm.DB
	log   *log.Helper
	cfg   *conf.Data
	redis *redis.Client
}

// NewData
func NewData(c *conf.Data, logger log.Logger, db *gorm.DB, rdb *redis.Client) (*Data, error) {
	data := &Data{
		log:   log.NewHelper(logger),
		db:    db,
		redis: rdb,
		cfg:   c,
	}

	return data, nil
}

// NewDB 连接 MySQL 数据库并自动迁移
func NewDB(c *conf.Data) (*gorm.DB, error) {
	if c == nil {
		return nil, fmt.Errorf("config data is nil")
	}

	dsn := c.Database.Source

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	if err = db.AutoMigrate(&DO.Seat{}, &DO.TimeSlot{}, &DO.Comment{}, &DO.FutureRecord{}, &DO.HistoryRecord{}, &DO.CreditSummary{}, &DO.CreditRecord{}, &DO.Record{}); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	return db, nil
}

// NewRedisDB 连接redis
func NewRedisDB(c *conf.Data, logger log.Logger) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		ReadTimeout:  time.Duration(c.Redis.ReadTimeout.GetSeconds()) * time.Second,
		WriteTimeout: time.Duration(c.Redis.WriteTimeout.GetSeconds()) * time.Second,
		DB:           0,
		Password:     c.Redis.Password,
	})
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic(fmt.Sprintf("connect redis err:%v", err))
	}
	log.NewHelper(logger).Info("redis connect success")
	return rdb
}
