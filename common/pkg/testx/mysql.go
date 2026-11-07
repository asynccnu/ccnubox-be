package testx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MySQL 在显式配置后为每个测试创建独立数据库，绝不迁移或清空 DSN 指向的业务库。
// 未配置时返回 nil，由调用方继续使用原有 SQLite 单元测试。
func MySQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("CCNUBOX_TEST_MYSQL_DSN")
	if dsn == "" {
		return nil
	}
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal("测试 MySQL DSN 无效")
	}
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 15 * time.Second
	cfg.WriteTimeout = 15 * time.Second
	options := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	admin, err := gorm.Open(mysql.Open(cfg.FormatDSN()), options)
	if err != nil {
		t.Fatal("测试 MySQL 管理连接失败，未输出连接凭据")
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminSQL.Close() })
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	name := "ccnubox_test_" + hex.EncodeToString(random[:])
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = admin.WithContext(ctx).Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin").Error
	cancel()
	if err != nil {
		t.Fatal("创建独立测试库失败，请确认 CREATE DATABASE 权限")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := admin.WithContext(ctx).Exec("DROP DATABASE `" + name + "`").Error; err != nil {
			t.Errorf("清理独立测试库失败: %s", name)
		}
	})
	cfg.DBName = name
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), options)
	if err != nil {
		t.Fatal("独立测试库连接失败，未输出连接凭据")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	t.Cleanup(func() { _ = sqlDB.Close() })
	t.Logf("独立 MySQL 测试库: %s", name)
	return db
}
