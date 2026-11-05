package dao

import (
	"fmt"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"gorm.io/gorm"
)

func InitTables(db *gorm.DB) error {
	if db.Migrator().HasTable(&model.Grade{}) {
		// 已有成绩表只检查结构，不扫描存量或在启动时隐式执行大表 DDL。
		if err := checkGradeNotificationSchema(db); err != nil {
			return err
		}
	} else if err := db.Migrator().CreateTable(&model.Grade{}); err != nil {
		return err
	}
	return db.AutoMigrate(&model.Rank{}, &model.GradeIdentityAlias{},
		&model.GradeSyncState{}, &model.GradeNotificationOutbox{}, &model.GradeSyncIssue{})
}

func gradeNotificationColumns(db *gorm.DB) (map[string]gorm.ColumnType, error) {
	columns, err := db.Migrator().ColumnTypes(&model.Grade{})
	if err != nil {
		return nil, err
	}
	result := map[string]gorm.ColumnType{}
	for _, column := range columns {
		result[column.Name()] = column
	}
	return result, nil
}

func notificationSchemaError(reason string) error {
	return fmt.Errorf("成绩通知需要 DDL 迁移，请先执行 be-grade/cmd/migrate-notifications: %s", reason)
}

func checkGradeNotificationColumns(db *gorm.DB, columns map[string]gorm.ColumnType, requireNullable bool) error {
	identity, hasIdentity := columns["notification_id"]
	version, hasVersion := columns["score_version"]
	if !hasIdentity || !hasVersion {
		return notificationSchemaError("缺少身份或业务版本字段")
	}
	length, lengthKnown := identity.Length()
	if !strings.EqualFold(identity.DatabaseTypeName(), "varchar") || !lengthKnown || length != 32 {
		return notificationSchemaError("notification_id 必须为 varchar(32)")
	}
	if nullable, known := identity.Nullable(); !known || (requireNullable && !nullable) {
		return notificationSchemaError("notification_id 必须允许 NULL")
	}
	if _, present := identity.DefaultValue(); present {
		return notificationSchemaError("notification_id 不能设置非 NULL 默认值")
	}
	kind := strings.ToLower(version.DatabaseTypeName())
	if kind != "bigint" && !(db.Dialector.Name() == "sqlite" && kind == "integer") {
		return notificationSchemaError("score_version 必须为 bigint")
	}
	if nullable, known := version.Nullable(); !known || nullable {
		return notificationSchemaError("score_version 必须非空")
	}
	value, present := version.DefaultValue()
	if !present || strings.Trim(value, "'\"() ") != "0" {
		return notificationSchemaError("score_version 默认值必须为 0")
	}
	return nil
}

func hasGradeNotificationUniqueIndex(db *gorm.DB) (bool, error) {
	indexes, err := db.Migrator().GetIndexes(&model.Grade{})
	if err != nil {
		return false, err
	}
	for _, index := range indexes {
		unique, known := index.Unique()
		if columns := index.Columns(); known && unique && len(columns) == 1 && columns[0] == "notification_id" {
			return true, nil
		}
	}
	return false, nil
}

func checkGradeNotificationSchema(db *gorm.DB) error {
	columns, err := gradeNotificationColumns(db)
	if err != nil {
		return err
	}
	if err := checkGradeNotificationColumns(db, columns, true); err != nil {
		return err
	}
	unique, err := hasGradeNotificationUniqueIndex(db)
	if err != nil {
		return err
	}
	if !unique {
		return notificationSchemaError("缺少 notification_id 单列唯一索引")
	}
	return nil
}
