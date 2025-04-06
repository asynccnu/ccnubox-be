package test

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
)

func NewDB(addr string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(addr), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect mysql failed:%v", err))
	}
	log.Info("connect mysql successfully")
	return db
}

func NewRedisDB(addr string, password string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})
	_, err := rdb.Ping().Result()
	if err != nil {
		panic(fmt.Sprintf("connect redis err:%v", err))
	}
	log.Info("connect redis successfully")
	return rdb
}

func NewLogger() log.Logger {
	return log.NewStdLogger(os.Stdout)
}
