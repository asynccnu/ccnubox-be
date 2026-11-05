package model

import (
	"crypto/rand"
	"encoding/hex"
)

const (
	NotificationPending   = "pending"
	NotificationPublished = "published"
	NotificationInvalid   = "invalid"
)

func NewNotificationID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(id[:]), nil
}

func ValidNotificationID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

type GradeIdentityAlias struct {
	StudentID      string `gorm:"type:varchar(100);not null;primaryKey"`
	SourceSystem   string `gorm:"type:varchar(32);not null;primaryKey"`
	AliasKind      string `gorm:"type:varchar(32);not null;primaryKey"`
	AliasValue     string `gorm:"type:varchar(100);not null;primaryKey"`
	NotificationID string `gorm:"type:varchar(32);not null;index"`
}

type GradeSyncState struct {
	StudentID     string `gorm:"type:varchar(100);primaryKey"`
	InitializedAt int64  `gorm:"not null;default:0"`
	LastSuccessAt int64  `gorm:"not null;default:0"`
}

type GradeNotificationOutbox struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	DedupeKey      string `gorm:"type:varchar(80);not null;uniqueIndex"`
	StudentID      string `gorm:"type:varchar(100);not null;index"`
	NotificationID string `gorm:"type:varchar(32);not null"`
	ScoreVersion   int64  `gorm:"not null"`
	Payload        string `gorm:"type:text;not null"`
	ObservedAt     int64  `gorm:"not null"`
	Status         string `gorm:"type:varchar(16);not null;index:idx_grade_notification_due,priority:1"`
	Attempts       int64  `gorm:"not null;default:0"`
	NextAttemptAt  int64  `gorm:"not null;index:idx_grade_notification_due,priority:2"`
	LastError      string `gorm:"type:varchar(512);not null"`
	CreatedAt      int64  `gorm:"autoCreateTime"`
	UpdatedAt      int64  `gorm:"autoUpdateTime"`
}

// 含身份歧义的快照保留待人工核验；仅跳过歧义课程，不影响其他课程同步。
type GradeSyncIssue struct {
	StudentID string `gorm:"type:varchar(100);primaryKey"`
	Payload   string `gorm:"type:longtext;not null"`
	LastError string `gorm:"type:varchar(512);not null"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}

func (GradeNotificationOutbox) TableName() string { return "grade_notification_outbox" }
