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
func TestInsertRefreshLog(t *testing.T) {
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

	ctx := context.Background()

	// 定义测试用例
	tests := []struct {
		name           string
		setup          func() // 用于设置初始数据
		stuID          string
		year           string
		semester       string
		expectedError  bool
		expectedStatus string
	}{
		{
			name: "Insert new log",
			setup: func() {
				// 无需设置初始数据
			},
			stuID:          "123456",
			year:           "2025",
			semester:       "1",
			expectedError:  false,
			expectedStatus: model.Pending,
		},
		{
			name: "Insert duplicate log within interval",
			setup: func() {
				db.Create(&model.ClassRefreshLog{
					StuID:     "123456",
					Year:      "2025",
					Semester:  "1",
					Status:    model.Pending,
					UpdatedAt: time.Now(),
				})
			},
			stuID:          "123456",
			year:           "2025",
			semester:       "1",
			expectedError:  true,
			expectedStatus: "",
		},
		{
			name: "Insert log after interval",
			setup: func() {
				// 临时禁用 BeforeCreate 钩子
				db = db.Session(&gorm.Session{SkipHooks: true})
				db.Create(&model.ClassRefreshLog{
					StuID:     "123456",
					Year:      "2025",
					Semester:  "1",
					Status:    model.Pending,
					UpdatedAt: time.Now().Add(-10 * time.Minute),
				})
			},
			stuID:          "123456",
			year:           "2025",
			semester:       "1",
			expectedError:  false,
			expectedStatus: model.Pending,
		},
		{
			name: "Insert log with failed status",
			setup: func() {
				db.Create(&model.ClassRefreshLog{
					StuID:     "123456",
					Year:      "2025",
					Semester:  "1",
					Status:    model.Failed,
					UpdatedAt: time.Now(),
				})
			},
			stuID:          "123456",
			year:           "2025",
			semester:       "1",
			expectedError:  false,
			expectedStatus: model.Pending,
		},
	}

	// 执行测试用例
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清空数据库
			db.Exec(`DELETE FROM ` + model.ClassRefreshLogTableName)

			// 设置初始数据
			tt.setup()

			// 调用 InsertRefreshLog
			logID, err := repo.InsertRefreshLog(ctx, tt.stuID, tt.year, tt.semester)

			// 验证结果
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, logID)

				// 验证插入的记录
				var log model.ClassRefreshLog
				err = db.First(&log, logID).Error
				assert.NoError(t, err)
				assert.Equal(t, tt.stuID, log.StuID)
				assert.Equal(t, tt.year, log.Year)
				assert.Equal(t, tt.semester, log.Semester)
				assert.Equal(t, tt.expectedStatus, log.Status)
			}
		})
	}
}
