package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// GradeNotificationKey 只依赖成绩身份与业务版本，不受元数据、详情或抓取时间影响。
func GradeNotificationKey(studentID, notificationID string, version int64) string {
	identity, _ := json.Marshal([]any{studentID, notificationID, version})
	sum := sha256.Sum256(identity)
	return "grade:v2:" + hex.EncodeToString(sum[:])
}

// ParseGradeNotification 校验持久化事件与业务身份一致，损坏数据不能被当作新事件重建。
func ParseGradeNotification(row model.GradeNotificationOutbox) (*feedv1.FeedEvent, error) {
	event := &feedv1.FeedEvent{}
	if err := protojson.Unmarshal([]byte(row.Payload), event); err != nil {
		return nil, errors.New("invalid grade notification payload")
	}
	if !model.ValidNotificationID(row.NotificationID) {
		return nil, errors.New("invalid grade notification identity")
	}
	key := GradeNotificationKey(row.StudentID, row.NotificationID, row.ScoreVersion)
	reason := event.ExtendFields["change_reason"]
	if strings.TrimSpace(row.StudentID) == "" || len(row.StudentID) > 64 || len(row.NotificationID) != 32 || row.ScoreVersion < 1 ||
		len(event.Title) > 1024 || len(event.Content) > 8192 || len(event.Url) > 1024 || len(event.ExtendFields) != 4 ||
		row.DedupeKey != key || event.DedupeKey != key || event.Type != feedv1.FeedEventType_GRADE || event.Source != "grade" ||
		row.ObservedAt <= 0 || event.OccurredAt != row.ObservedAt || event.Title == "" || event.Content == "" ||
		!strings.HasPrefix(event.Url, "ccnubox://scoreCalculation?") || event.ExtendFields["url"] != event.Url ||
		event.ExtendFields["notification_id"] != row.NotificationID || event.ExtendFields["score_version"] != strconv.FormatInt(row.ScoreVersion, 10) ||
		(reason != "first_published" && reason != "score_changed") {
		return nil, errors.New("grade notification identity or metadata mismatch")
	}
	return event, nil
}

// NewGradeNotification 在成绩事务中生成不可变快照，发布器不重新读取课程信息。
func NewGradeNotification(g model.Grade, reason string, observedAt int64) (model.GradeNotificationOutbox, error) {
	if !g.ValidNotificationState() || g.NotificationID == nil {
		return model.GradeNotificationOutbox{}, errors.New("grade notification identity is not activated")
	}
	id := g.NotificationIdentity()
	values := url.Values{}
	semester, _ := json.Marshal([]string{fmt.Sprintf("%d-%d", g.Xnm, g.Xqm)})
	courseType, _ := json.Marshal([]string{g.Kcxzmc})
	values.Set("semester", string(semester))
	values.Set("courseType", string(courseType))
	link := "ccnubox://scoreCalculation?" + values.Encode()
	title, content := "成绩发布提醒", fmt.Sprintf("您的课程:%s成绩已发布,请及时查看", g.Kcmc)
	if reason == "score_changed" {
		title, content = "成绩更新提醒", fmt.Sprintf("您的课程:%s分数更新了,请及时查看", g.Kcmc)
	}
	key := GradeNotificationKey(g.StudentId, id, g.ScoreVersion)
	payload, err := protojson.Marshal(&feedv1.FeedEvent{
		Type: feedv1.FeedEventType_GRADE, Source: "grade", Title: title, Content: content,
		Url: link, DedupeKey: key, OccurredAt: observedAt,
		ExtendFields: map[string]string{
			"url": link, "notification_id": id,
			"score_version": strconv.FormatInt(g.ScoreVersion, 10), "change_reason": reason,
		},
	})
	if err != nil {
		return model.GradeNotificationOutbox{}, err
	}
	row := model.GradeNotificationOutbox{
		DedupeKey: key, StudentID: g.StudentId, NotificationID: id,
		ScoreVersion: g.ScoreVersion, Payload: string(payload), ObservedAt: observedAt,
		Status: model.NotificationPending, NextAttemptAt: observedAt,
		CreatedAt: observedAt, UpdatedAt: observedAt,
	}
	if _, err := ParseGradeNotification(row); err != nil {
		return model.GradeNotificationOutbox{}, err
	}
	return row, nil
}
