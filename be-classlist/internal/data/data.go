package data

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	logger2 "gorm.io/gorm/logger"
	logger3 "log"
	"os"
	"time"
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
)

// Data .
type Data struct {
	Mysql *gorm.DB
}

// NewData .
func NewData(c *conf.Data, mysqlDB *gorm.DB, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	return &Data{
		Mysql: mysqlDB,
	}, cleanup, nil
}

// NewDB 连接mysql数据库
func NewDB(c *conf.Data, logfile *os.File, logger log.Logger) *gorm.DB {

	var logLevel map[string]logger2.LogLevel
	logLevel = map[string]logger2.LogLevel{
		"info":  logger2.Info,
		"warn":  logger2.Warn,
		"error": logger2.Error,
	}

	level, ok := logLevel[c.Database.LogLevel]
	if !ok {
		level = logger2.Warn
	}

	//注意:
	//这个logfile 最好别在此处声明,最好在main函数中声明,在程序结束时关闭
	//否则你只能在下面的db.AutoMigrate得到相关日志
	newlogger := logger2.New(
		//日志写入文件
		logger3.New(logfile, "\r\n", logger3.LstdFlags),
		logger2.Config{
			SlowThreshold: time.Second,
			LogLevel:      level,
			Colorful:      false,
		},
	)

	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{Logger: newlogger})
	if err != nil {
		panic(fmt.Sprintf("connect mysql failed:%v", err))
	}
	if err := db.AutoMigrate(&model.ClassInfo{}, &model.StudentCourse{}, &model.Jxb{}, &model.ClassRefreshLog{}); err != nil {
		panic(fmt.Sprintf("mysql auto migrate failed:%v", err))
	}

	log.NewHelper(logger).Info("mysql connect success")

	return db
}

// NewRedisDB 连接redis
func NewRedisDB(c *conf.Data, logger log.Logger) *redis.Client {
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
	log.NewHelper(logger).Info("redis connect success")
	return rdb
}

func initProducerConfig() *sarama.Config {
	producerConfig := sarama.NewConfig()
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

func initConsumerConfig() *sarama.Config {
	consumerConfig := sarama.NewConfig()
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerConfig.Consumer.Group.Session.Timeout = 10 * time.Second
	consumerConfig.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	return consumerConfig
}

type KafkaProducerBuilder struct {
	brokers []string
}

func NewKafkaProducerBuilder(c *conf.Data) *KafkaProducerBuilder {
	return &KafkaProducerBuilder{
		brokers: c.Kafka.Brokers,
	}
}

func (pb KafkaProducerBuilder) Build() (sarama.SyncProducer, error) {
	producerConfig := initProducerConfig()
	p, err := sarama.NewSyncProducer(pb.brokers, producerConfig)
	if err != nil {
		return nil, fmt.Errorf("kafka producer connect failed: %w", err)
	}
	return p, nil
}

type KafkaConsumerBuilder struct {
	brokers []string
}

func NewKafkaConsumerBuilder(c *conf.Data) *KafkaConsumerBuilder {
	return &KafkaConsumerBuilder{
		brokers: c.Kafka.Brokers,
	}
}

func (cb KafkaConsumerBuilder) Build(groupID string) (sarama.ConsumerGroup, error) {
	consumerConfig := initConsumerConfig()
	consumerGroup, err := sarama.NewConsumerGroup(cb.brokers, groupID, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer connect failed: %w", err)
	}
	return consumerGroup, nil
}
