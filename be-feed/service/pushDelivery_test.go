package service

import (
	"strconv"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
)

func TestLibraryPushExpired(t *testing.T) {
	// 固定时钟，精确覆盖截止前一秒、截止时刻和截止后一秒。
	now := time.Unix(1700000000, 0)
	future := strconv.FormatInt(now.Unix()+1, 10)
	past := strconv.FormatInt(now.Unix()-1, 10)

	reminders := []struct {
		name             string
		eventType        string
		notificationType string
		field            string
	}{
		{"START_30", "library", "START_30", "start_at"},
		{"END_10", "library", "END_10", "end_at"},
		{"AWAY_60", "library", "AWAY_60", "end_at"},
		{"AWAY_80", "library", "AWAY_80", "end_at"},
		{"normalized_START_30", "LiBrArY", " \tstart_30\n", "start_at"},
		{"normalized_END_10", "LiBrArY", " \tend_10\n", "end_at"},
		{"normalized_AWAY_60", "LiBrArY", " \taway_60\n", "end_at"},
		{"normalized_AWAY_80", "LiBrArY", " \taway_80\n", "end_at"},
	}
	deadlines := []struct {
		name    string
		value   string
		missing bool
		want    bool
	}{
		{name: "future", value: future},
		{name: "at_deadline", value: strconv.FormatInt(now.Unix(), 10), want: true},
		{name: "past", value: past, want: true},
		{name: "missing", missing: true, want: true},
		{name: "empty", value: "", want: true},
		{name: "invalid", value: "invalid", want: true},
		{name: "overflow", value: "9223372036854775808", want: true},
		{name: "zero", value: "0", want: true},
		{name: "negative", value: "-1", want: true},
	}
	for _, reminder := range reminders {
		t.Run(reminder.name, func(t *testing.T) {
			for _, deadline := range deadlines {
				t.Run(deadline.name, func(t *testing.T) {
					// 另一时间字段与预期相反，防止误选字段或回退到另一字段。
					other := future
					if !deadline.want {
						other = past
					}
					fields := model.ExtendFields{
						"notification_type": reminder.notificationType,
						"start_at":          other,
						"end_at":            other,
					}
					if deadline.missing {
						delete(fields, reminder.field)
					} else {
						fields[reminder.field] = deadline.value
					}
					event := model.FeedEvent{Type: reminder.eventType, ExtendFields: fields}
					if got := libraryPushExpired(&event, now); got != deadline.want {
						t.Fatalf("libraryPushExpired(%+v) = %v, want %v", event, got, deadline.want)
					}
				})
			}
		})
	}

	// 普通通知、事实类消息及未知通知类型不受时间字段限制。
	unaffected := []struct {
		name             string
		eventType        string
		notificationType string
	}{
		{"ordinary", "muxi", ""},
		{"ordinary_START_30", "muxi", "START_30"},
		{"ordinary_END_10", "muxi", "END_10"},
		{"ordinary_AWAY_60", "muxi", "AWAY_60"},
		{"ordinary_AWAY_80", "muxi", "AWAY_80"},
		{"RESERVATION_DISCOVERED", "library", "RESERVATION_DISCOVERED"},
		{"BREACH", "library", "BREACH"},
		{"BLACKLISTED", "library", "BLACKLISTED"},
		{"unknown", "library", "UNKNOWN"},
		{"missing_notification_type", "library", ""},
	}
	for _, tt := range unaffected {
		t.Run(tt.name, func(t *testing.T) {
			for _, deadline := range deadlines {
				t.Run(deadline.name, func(t *testing.T) {
					fields := model.ExtendFields{}
					if tt.notificationType != "" {
						fields["notification_type"] = tt.notificationType
					}
					if !deadline.missing {
						fields["start_at"] = deadline.value
						fields["end_at"] = deadline.value
					}
					event := model.FeedEvent{Type: tt.eventType, ExtendFields: fields}
					if libraryPushExpired(&event, now) {
						t.Fatalf("libraryPushExpired(%+v) = true, want false", event)
					}
				})
			}
		})
	}
}
