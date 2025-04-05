package dao

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type StaticDAO interface {
	GetStaticByName(ctx context.Context, name string) (Static, error)
	Upsert(ctx context.Context, static Static) error
	GetStaticsByLabels(ctx context.Context, labels map[string]string) ([]Static, error)
}

type MongoDBStaticDAO struct {
	staticCol *mongo.Collection
}

func NewMongoDBStaticDAO(db *mongo.Database) StaticDAO {
	return &MongoDBStaticDAO{staticCol: db.Collection("statics")}
}

func (dao *MongoDBStaticDAO) GetStaticByName(ctx context.Context, name string) (Static, error) {
	var s Static
	filter := bson.M{"name": name}
	err := dao.staticCol.FindOne(ctx, filter).Decode(&s)
	return s, err
}

func (dao *MongoDBStaticDAO) Upsert(ctx context.Context, static Static) error {
	filter := bson.M{"name": static.Name}
	update := bson.M{
		"$set": bson.M{
			"content": static.Content,
			"labels":  static.Labels,
			"utime":   time.Now().Format(time.DateTime),
		},
	}
	_, err := dao.staticCol.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func (dao *MongoDBStaticDAO) GetStaticsByLabels(ctx context.Context, labels map[string]string) ([]Static, error) {
	filter := make(bson.M, 5)
	for key, val := range labels {
		filter["labels."+key] = val
	}
	cursor, err := dao.staticCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var res []Static
	err = cursor.All(ctx, &res)
	return res, err
}

type Static struct {
	Name    string            `bson:"name"` // 这个是唯一索引映射，我忽略掉了MongoDB的_id
	Content string            `bson:"content"`
	Utime   string            `bson:"utime"`
	Labels  map[string]string `bson:"labels"`
}
