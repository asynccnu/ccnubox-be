package ioc

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-static/repository/dao"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

func InitDB() *mongo.Database {
	type Config struct {
		URI string `yaml:"uri"`
		DB  string `yaml:"db"`
	}
	var cfg Config
	err := viper.UnmarshalKey("mongodb", &cfg)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		panic(err)
	}
	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		panic(err)
	}
	db := client.Database(cfg.DB)
	err = dao.InitCollections(db)
	if err != nil {
		panic(err)
	}
	return db
}
