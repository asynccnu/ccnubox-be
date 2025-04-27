package data_test

import (
	"context"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateRefreshLogStatus(t *testing.T) {
	// 初始化内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// 自动建表
	err = db.AutoMigrate(&model.ClassRefreshLog{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// 初始化 RefreshLogRepo
	cf := &conf.Server{RefreshInterval: 60}
	repo := data.NewRefreshLogRepo(db, cf)

	// 插入一条初始数据
	initialLog := model.ClassRefreshLog{
		StuID:     "123456",
		Year:      "2025",
		Semester:  "1",
		Status:    model.Pending,
		UpdatedAt: time.Now(),
	}
	if err := db.Create(&initialLog).Error; err != nil {
		t.Fatalf("failed to insert initial refresh log: %v", err)
	}

	// 调用 UpdateRefreshLogStatus 将状态更新为 Ready
	newStatus := model.Ready
	err = repo.UpdateRefreshLogStatus(context.Background(), initialLog.ID, newStatus)
	assert.NoError(t, err)

	// 验证更新后的结果
	var updatedLog model.ClassRefreshLog
	err = db.First(&updatedLog, initialLog.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, newStatus, updatedLog.Status)

	// 再次调用 UpdateRefreshLogStatus 将状态更新为 Failed
	failedStatus := model.Failed
	err = repo.UpdateRefreshLogStatus(context.Background(), initialLog.ID, failedStatus)
	assert.NoError(t, err)

	// 验证更新后的结果
	err = db.First(&updatedLog, initialLog.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, failedStatus, updatedLog.Status)
}
