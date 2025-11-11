package data

import (
	"context"
	"errors"
	"github.com/asynccnu/ccnubox-be/be-class/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-class/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	logger2 "gorm.io/gorm/logger"
	"io"
	logger3 "log"
	"time"
)

var (
	ErrRecordNotFound = biz.ErrRecordNotFound
)

type CultivateStrategyData struct {
	db        *gorm.DB
	dataAlive time.Duration
}

func NewCultivateStrategyData(db *gorm.DB, c *conf.Data) biz.CultivateStrategyData {
	alive := time.Duration(c.Database.DataAlive) * time.Hour * 24
	return &CultivateStrategyData{
		db:        db,
		dataAlive: alive,
	}
}

func NewDB(c *conf.Data, logfile io.Writer) *gorm.DB {
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

	newlogger := logger2.New(
		logger3.New(logfile, "\r\n", logger3.LstdFlags),
		logger2.Config{
			SlowThreshold: time.Second,
			LogLevel:      level,
			Colorful:      false,
		},
	)

	db, err := gorm.Open(mysql.Open(c.Database.Dsn), &gorm.Config{
		Logger: newlogger,
	})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(&model.UnStudiedClassStudentRelationship{}, &model.ToBeStudiedClass{})
	if err != nil {
		panic("failed to create table")
	}

	return db
}

func (c *CultivateStrategyData) BatchSaveToBeStudiedClass(ctx context.Context,
	relations []model.UnStudiedClassStudentRelationship, classes []model.ToBeStudiedClass) error {
	tx := c.db.Begin()

	now := time.Now().Unix()
	for i := range relations {
		relations[i].UpdatedAt = now
	}
	if err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "to_be_studied_class_id"}, {Name: "student_id"}}, // 这里联合差重
			DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
		}).Create(&relations).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(&classes).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (c *CultivateStrategyData) GetClassStudentRelation(ctx context.Context, stuId, status string,
	dataAlive time.Duration) ([]model.UnStudiedClassStudentRelationship, error) {
	var result []model.UnStudiedClassStudentRelationship
	q := c.db.Model(&model.UnStudiedClassStudentRelationship{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if dataAlive > 0 {
		q = q.Where("updated_at > ?", time.Now().Add(-dataAlive))
	}

	if err := q.WithContext(ctx).Where("student_id = ?", stuId).Find(&result).Error; err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, ErrRecordNotFound
	}

	return result, nil
}

func (c *CultivateStrategyData) GetDetailUnStudyClass(ctx context.Context, id string) (model.ToBeStudiedClass, error) {
	var result model.ToBeStudiedClass
	if err := c.db.Model(&model.ToBeStudiedClass{}).WithContext(ctx).Where("id = ?", id).First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ToBeStudiedClass{}, ErrRecordNotFound
		}
		return model.ToBeStudiedClass{}, err
	}

	return result, nil
}

func (c *CultivateStrategyData) DataAlive() time.Duration {
	return c.dataAlive
}
