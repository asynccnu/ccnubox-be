package data

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data/do"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	glog "gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewDB,
	NewRedisDB,
	NewStudentAndCourseDBRepo,
	NewStudentAndCourseCacheRepo,
	NewClassInfoDBRepo,
	NewClassInfoCacheRepo,
	NewJxbDBRepo,
	NewRefreshLogRepo,
	NewKafkaProducerBuilder,
	NewKafkaConsumerBuilder,
	NewDelayKafkaConfig,
	NewDelayKafka,
	NewClassInfoRepo,
	NewStudentAndCourseRepo,
	NewClassRepo,
	NewLogger,
	NewKratosLogger,
	NewGromLogger,
)

type Transaction interface {
	// 下面2个方法配合使用，在InTx方法中执行ORM操作的时候需要使用DB方法获取db！
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	DB(ctx context.Context) *gorm.DB
}

// Data .
type Data struct {
	Mysql *gorm.DB
}

// NewData .
func NewData(c *conf.Data, mysqlDB *gorm.DB, logger logger.Logger) (*Data, func(), error) {
	cleanup := func() {
		logger.Info("closing the data resources")
	}
	return &Data{
		Mysql: mysqlDB,
	}, cleanup, nil
}

// NewDB 连接mysql数据库
func NewDB(c *conf.Data, logger logger.Logger, glogger glog.Interface) *gorm.DB {
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{Logger: glogger})
	if err != nil {
		panic(fmt.Sprintf("connect mysql failed:%v", err))
	}
	if err := db.AutoMigrate(
		&do.ClassInfo{},
		&do.StudentCourse{},
		&do.Jxb{},
		&do.ClassRefreshLog{},
	); err != nil {
		panic(fmt.Sprintf("mysql auto migrate failed:%v", err))
	}

	logger.Info("mysql connect success")

	return db
}

// NewRedisDB 连接redis
func NewRedisDB(c *conf.Data, logger logger.Logger) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		ReadTimeout:  time.Duration(c.Redis.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(c.Redis.WriteTimeout) * time.Millisecond,
		DB:           0,
		Password:     c.Redis.Password,
	})
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic(fmt.Sprintf("connect redis err:%v", err))
	}
	logger.Info("redis connect success")
	return rdb
}

func initProducerConfig(username, password string) *sarama.Config {
	producerConfig := sarama.NewConfig()
	producerConfig.Net.SASL.Enable = true
	producerConfig.Net.SASL.User = username
	producerConfig.Net.SASL.Password = password
	producerConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	producerConfig.Producer.Return.Errors = true
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Partitioner = sarama.NewHashPartitioner
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.MaxMessageBytes = 1000000
	producerConfig.Producer.Timeout = 10 * time.Second
	producerConfig.Producer.Retry.Max = 3
	producerConfig.Producer.Retry.Backoff = 100 * time.Millisecond
	producerConfig.Producer.CompressionLevel = sarama.CompressionLevelDefault
	return producerConfig
}

func initConsumerConfig(username, password string) *sarama.Config {
	consumerConfig := sarama.NewConfig()
	consumerConfig.Net.SASL.Enable = true
	consumerConfig.Net.SASL.User = username
	consumerConfig.Net.SASL.Password = password
	consumerConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext

	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Group.Session.Timeout = 10 * time.Second
	consumerConfig.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	return consumerConfig
}

type KafkaProducerBuilder struct {
	brokers  []string
	username string
	password string
}

func NewKafkaProducerBuilder(c *conf.Data) *KafkaProducerBuilder {
	return &KafkaProducerBuilder{
		brokers:  c.Kafka.Brokers,
		username: c.Kafka.Username,
		password: c.Kafka.Password,
	}
}

func (pb KafkaProducerBuilder) Build() (sarama.SyncProducer, error) {
	producerConfig := initProducerConfig(pb.username, pb.password)
	p, err := sarama.NewSyncProducer(pb.brokers, producerConfig)
	if err != nil {
		return nil, fmt.Errorf("kafka producer connect failed: %w", err)
	}
	return p, nil
}

type KafkaConsumerBuilder struct {
	brokers  []string
	username string
	password string
}

func NewKafkaConsumerBuilder(c *conf.Data) *KafkaConsumerBuilder {
	return &KafkaConsumerBuilder{
		brokers:  c.Kafka.Brokers,
		username: c.Kafka.Username,
		password: c.Kafka.Password,
	}
}

func (cb KafkaConsumerBuilder) Build(groupID string) (sarama.ConsumerGroup, error) {
	consumerConfig := initConsumerConfig(cb.username, cb.password)
	consumerGroup, err := sarama.NewConsumerGroup(cb.brokers, groupID, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer connect failed: %w", err)
	}
	return consumerGroup, nil
}
