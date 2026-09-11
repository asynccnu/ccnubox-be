package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// MaxFeedEventURLBytes 保留给 gRPC 的保守字节限制。
	MaxFeedEventURLBytes = 2047

	MaxFeedEventTypeChars         = 255
	MaxFeedEventStudentIDChars    = 255
	MaxFeedEventURLChars          = 2047
	MaxFeedEventDedupeKeyChars    = 255
	MaxFeedEventSourceChars       = 64
	MaxFeedEventTextBytes         = 65535
	MaxFeedEventExtendFieldsBytes = 65535
)

// FeedEventValidationError 表示重试无法修复的消息存储边界错误。
type FeedEventValidationError struct {
	Field  string
	Reason string
}

func (e *FeedEventValidationError) Error() string {
	return fmt.Sprintf("invalid feed event %s: %s", e.Field, e.Reason)
}

// ValidateFeedEventForStorage 校验 feed_events 及投递表会写入的全部有限字段。
func ValidateFeedEventForStorage(event FeedEvent) error {
	if err := validateRequiredChars("student_id", event.StudentId, MaxFeedEventStudentIDChars); err != nil {
		return err
	}
	if err := validateRequiredChars("type", event.Type, MaxFeedEventTypeChars); err != nil {
		return err
	}
	switch strings.ToLower(event.Type) {
	case "grade", "holiday", "muxi", "energy", "feedback", "library":
	default:
		return &FeedEventValidationError{Field: "type", Reason: "unsupported value"}
	}
	if err := validateText("title", event.Title); err != nil {
		return err
	}
	if err := validateText("content", event.Content); err != nil {
		return err
	}
	if err := validateOptionalChars("url", event.Url, MaxFeedEventURLChars); err != nil {
		return err
	}
	if err := validateRequiredChars("dedupe_key", event.DedupeKey, MaxFeedEventDedupeKeyChars); err != nil {
		return err
	}
	if err := validateOptionalChars("source", event.Source, MaxFeedEventSourceChars); err != nil {
		return err
	}
	for key, value := range event.ExtendFields {
		if !utf8.ValidString(key) || !utf8.ValidString(value) {
			return &FeedEventValidationError{Field: "extend_fields", Reason: "contains invalid UTF-8"}
		}
	}
	raw, err := json.Marshal(event.ExtendFields)
	if err != nil {
		return &FeedEventValidationError{Field: "extend_fields", Reason: "cannot encode JSON"}
	}
	if len(raw) > MaxFeedEventExtendFieldsBytes {
		return &FeedEventValidationError{Field: "extend_fields", Reason: "encoded JSON exceeds TEXT limit"}
	}
	return nil
}

// KafkaFeedEventDedupeKey 为旧的空 key 消息生成可跨 Kafka 重投复用的接收者级 ID。
func KafkaFeedEventDedupeKey(topic string, partition int32, offset int64, studentID string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%d\x00%s", topic, partition, offset, studentID)))
	return "kafka:" + hex.EncodeToString(sum[:])
}

func validateRequiredChars(field, value string, maxChars int) error {
	if strings.TrimSpace(value) == "" {
		return &FeedEventValidationError{Field: field, Reason: "is required"}
	}
	return validateOptionalChars(field, value, maxChars)
}

func validateOptionalChars(field, value string, maxChars int) error {
	if !utf8.ValidString(value) {
		return &FeedEventValidationError{Field: field, Reason: "contains invalid UTF-8"}
	}
	if utf8.RuneCountInString(value) > maxChars {
		return &FeedEventValidationError{Field: field, Reason: fmt.Sprintf("exceeds %d characters", maxChars)}
	}
	return nil
}

func validateText(field, value string) error {
	if !utf8.ValidString(value) {
		return &FeedEventValidationError{Field: field, Reason: "contains invalid UTF-8"}
	}
	if len(value) > MaxFeedEventTextBytes {
		return &FeedEventValidationError{Field: field, Reason: "exceeds TEXT byte limit"}
	}
	return nil
}

// FeedEvent的模型
type FeedEvent struct {
	ID           int64             `json:"id"` // ID
	StudentId    string            `json:"student_id"`
	Type         string            `json:"type"`          // 类型
	Title        string            `json:"title"`         // 提示用的字段
	Content      string            `json:"content"`       // 正式文本
	Url          string            `json:"url"`           // 消息详情跳转路由
	ExtendFields map[string]string `json:"extend_fields"` // 拓展字段
	CreatedAt    int64             `json:"created_at"`    // 创建时间，Unix 时间戳（int格式）
	DedupeKey    string            `json:"dedupe_key,omitempty"`
	Source       string            `json:"source,omitempty"`
	OccurredAt   int64             `json:"occurred_at,omitempty"`
}

// AllowList 表示更改推送消息数量的请求
type AllowList struct {
	StudentId string `json:"student_id"`
	Grade     bool   `json:"grade"`
	Muxi      bool   `json:"muxi"`
	Holiday   bool   `json:"holiday"`
	Energy    bool   `json:"energy"`
	FeedBack  bool   `json:"feed_back"`
	// 可空字段，兼容
	Library *bool `json:"library,omitempty"`
}

type LibraryPreferenceChange struct {
	Revision  int64
	StudentID string
	Enabled   bool
	ChangedAt int64
}

type LibraryReminderUser struct {
	ID        int64
	StudentID string
	Revision  int64
}

type MuxiOfficialMSG struct {
	Title        string
	Content      string
	ExtendFields       // 拓展字段如果要发额外的东西的话
	PublicTime   int64 // 正式发布的时间
	Id           string
}
