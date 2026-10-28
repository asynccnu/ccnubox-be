package dao

import (
	"fmt"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"gorm.io/gorm"
)

// 列名与类型均来自内部常量，不拼接请求参数；历史大小写差异统一在读取时兼容。
func feedTypeMaskSQL(column string) string {
	var expr strings.Builder
	fmt.Fprintf(&expr, "CASE HEX(LOWER(%s))", column)
	for _, label := range domain.FeedTypes() {
		position, _ := domain.FeedTypePosition(label)
		fmt.Fprintf(&expr, " WHEN '%X' THEN %d", []byte(label), 1<<position)
	}
	expr.WriteString(" ELSE 0 END")
	return expr.String()
}

// 关联时不能过滤配置的 deleted_at，否则会把软删除误判为缺失并默认开启。
func withFeedSubscription(query *gorm.DB, table string) *gorm.DB {
	return query.Joins("LEFT JOIN feed_user_configs AS subscription ON subscription.student_id = "+table+".student_id").
		Where("subscription.deleted_at IS NULL").
		Where("(CASE WHEN subscription.id IS NULL THEN ? ELSE subscription.push_config END & ("+feedTypeMaskSQL(table+".type")+")) <> 0", domain.MissingPushConfig)
}

// 旧失败消息仍由读取后软删除的路径消费，只删除当前允许返回的类型，保留关闭类型。
func withFailFeedSubscription(query *gorm.DB, studentID string) *gorm.DB {
	config := query.Session(&gorm.Session{NewDB: true}).Unscoped().Model(&model.FeedUserConfig{}).
		Select("CASE WHEN deleted_at IS NULL THEN push_config ELSE 0 END").Where("student_id = ?", studentID)
	return query.Where("(COALESCE((?), ?) & ("+feedTypeMaskSQL("feed_fail_events.type")+")) <> 0", config, domain.MissingPushConfig)
}
