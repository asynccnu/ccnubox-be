package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const gradeNotificationIdentityIndex = "uidx_grades_notification_id"

// MigrateGradeNotifications 只处理结构，不读取/回填历史成绩、别名、学生基线或 Outbox。
// 调用前必须备份并暂停成绩写入及旧发布入口；MySQL 5.7 加列仍可能重建整张表。
func MigrateGradeNotifications(ctx context.Context, db *gorm.DB) error {
	db = db.WithContext(ctx)
	if !db.Migrator().HasTable(&model.Grade{}) {
		return InitTables(db)
	}
	columns, err := gradeNotificationColumns(db)
	if err != nil {
		return err
	}
	_, hasIdentity := columns["notification_id"]
	_, hasVersion := columns["score_version"]
	if hasIdentity != hasVersion {
		return notificationSchemaError("字段只存在一部分，请保持停写并核对结构，不自动补齐")
	}
	if !hasIdentity {
		if db.Dialector.Name() == "mysql" {
			if err := alterGradeNotifications(db, "ADD COLUMN notification_id VARCHAR(32) NULL DEFAULT NULL, "+
				"ADD COLUMN score_version BIGINT NOT NULL DEFAULT 0, "+
				"ADD UNIQUE INDEX "+gradeNotificationIdentityIndex+" (notification_id)"); err != nil {
				return err
			}
		} else if db.Dialector.Name() == "sqlite" {
			for _, field := range []string{"NotificationID", "ScoreVersion"} {
				if err := db.Migrator().AddColumn(&model.Grade{}, field); err != nil {
					return err
				}
			}
			if err := db.Migrator().CreateIndex(&model.Grade{}, gradeNotificationIdentityIndex); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("成绩通知 DDL 不支持数据库类型 %s", db.Dialector.Name())
		}
	} else {
		if err := checkGradeNotificationColumns(db, columns, false); err != nil {
			return err
		}
		unique, err := hasGradeNotificationUniqueIndex(db)
		if err != nil {
			return err
		}
		if !unique {
			return notificationSchemaError("已有通知字段但缺少唯一索引，请保持停写并核对结构")
		}
		if nullable, _ := columns["notification_id"].Nullable(); !nullable {
			// 兼容已完成旧方案的环境，只放宽约束，不清空已分配身份或版本。
			if db.Dialector.Name() == "mysql" {
				if err := alterGradeNotifications(db, "MODIFY COLUMN notification_id VARCHAR(32) NULL DEFAULT NULL"); err != nil {
					return err
				}
			} else if db.Dialector.Name() == "sqlite" {
				if err := db.Migrator().AlterColumn(&model.Grade{}, "NotificationID"); err != nil {
					return err
				}
				unique, err := hasGradeNotificationUniqueIndex(db)
				if err != nil {
					return err
				}
				if !unique {
					if err := db.Migrator().CreateIndex(&model.Grade{}, gradeNotificationIdentityIndex); err != nil {
						return err
					}
				}
			} else {
				return fmt.Errorf("成绩通知 DDL 不支持数据库类型 %s", db.Dialector.Name())
			}
		}
	}
	return InitTables(db)
}

func alterGradeNotifications(db *gorm.DB, changes string) error {
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(&model.Grade{}); err != nil {
		return err
	}
	// 使用同一连接设置元数据锁等待上限，不能自动退回 COPY 或更强锁。
	return db.Connection(func(conn *gorm.DB) error {
		var oldTimeout int64
		if err := conn.Raw("SELECT @@session.lock_wait_timeout").Scan(&oldTimeout).Error; err != nil {
			return err
		}
		if err := conn.Exec("SET SESSION lock_wait_timeout = 10").Error; err != nil {
			return err
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = conn.WithContext(ctx).Exec("SET SESSION lock_wait_timeout = ?", oldTimeout).Error
		}()
		return conn.Exec("ALTER TABLE ? "+changes+", ALGORITHM=INPLACE, LOCK=NONE", clause.Table{Name: statement.Table}).Error
	})
}
