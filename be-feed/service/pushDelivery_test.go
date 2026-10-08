package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
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
		{"TEAM_INVITATION", "library", "TEAM_INVITATION", "expires_at"},
		{"normalized_invitation", "LiBrArY", " team_invitation ", "expires_at"},
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
		{"TEAM_SUCCESS", "library", "TEAM_SUCCESS"},
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

type invitationDeliveryDAO struct {
	dao.PushDeliveryDAO
	event *model.FeedEvent
	cid   string
	state string
}

func (d *invitationDeliveryDAO) Claim(context.Context, int64) (bool, error) { return true, nil }
func (d *invitationDeliveryDAO) GetFeedEvent(context.Context, int64) (*model.FeedEvent, error) {
	return d.event, nil
}
func (d *invitationDeliveryDAO) SaveCID(_ context.Context, _ int64, cid string) error {
	d.cid = cid
	return nil
}
func (d *invitationDeliveryDAO) MarkSent(context.Context, int64) error { d.state = "sent"; return nil }
func (d *invitationDeliveryDAO) MarkSuppressed(context.Context, int64) error {
	d.state = "suppressed"
	return nil
}
func (d *invitationDeliveryDAO) MarkRetry(context.Context, int64, int, int64, string, bool) error {
	d.state = "retry"
	return nil
}

type invitationDeliveryPush struct {
	PushService
	repo                *invitationDeliveryDAO
	expireDuringCID     bool
	err                 error
	cidCalls, pushCalls int
	seenCID             string
}

func (p *invitationDeliveryPush) PreparePush(context.Context, *domain.FeedEvent) (*PreparedPush, error) {
	return &PreparedPush{tokens: []string{"device"}}, nil
}
func (p *invitationDeliveryPush) GetPushCID(context.Context) (string, error) {
	p.cidCalls++
	if p.expireDuringCID {
		p.repo.event.ExtendFields["expires_at"] = "1"
	}
	return "fixed-cid", nil
}
func (p *invitationDeliveryPush) PushPreparedMSGWithCID(_ context.Context, _ *domain.FeedEvent, _ *PreparedPush, cid string) error {
	p.pushCalls++
	p.seenCID = cid
	return p.err
}

func TestInvitationDeliveryExpiresAndReusesCID(t *testing.T) {
	for _, tc := range []struct {
		name              string
		past, duringCID   bool
		pushErr           error
		wantState         string
		wantPush, wantCID int
	}{
		{name: "past", past: true, wantState: "suppressed"},
		{name: "crosses_deadline", duringCID: true, wantState: "suppressed", wantCID: 1},
		{name: "subsecond_sentinel", pushErr: jpush.ErrPushExpired, wantState: "suppressed", wantPush: 1, wantCID: 1},
		{name: "sent", wantState: "sent", wantPush: 1, wantCID: 1},
		{name: "retry_same_cid", pushErr: errors.New("network error"), wantState: "retry", wantPush: 1, wantCID: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deadline := time.Now().Add(time.Minute).Unix()
			if tc.past {
				deadline = 1
			}
			repo := &invitationDeliveryDAO{event: &model.FeedEvent{Type: "library", StudentId: "A", Source: "library", OccurredAt: 1, ExtendFields: model.ExtendFields{"notification_type": "TEAM_INVITATION", "team_id": "1", "expires_at": strconv.FormatInt(deadline, 10)}}}
			push := &invitationDeliveryPush{repo: repo, expireDuringCID: tc.duringCID, err: tc.pushErr}
			svc := &pushDeliveryService{dao: repo, push: push, log: zapx.NewZapLogger(zap.NewNop())}
			delivery := model.FeedPushDelivery{FeedEventID: 1}
			delivery.ID = 1
			if err := svc.dispatchOne(context.Background(), delivery); err != nil {
				t.Fatal(err)
			}
			if repo.state != tc.wantState || push.pushCalls != tc.wantPush || push.cidCalls != tc.wantCID {
				t.Fatalf("state=%s push=%d cid=%d", repo.state, push.pushCalls, push.cidCalls)
			}
			if tc.wantState == "retry" {
				delivery.CID = repo.cid
				push.err = nil
				if err := svc.dispatchOne(context.Background(), delivery); err != nil {
					t.Fatal(err)
				}
				if repo.state != "sent" || push.cidCalls != 1 || push.seenCID != "fixed-cid" {
					t.Fatal("CID changed on retry")
				}
			}
		})
	}
}

type invitationPushClient struct {
	jpush.PushClient
	data jpush.PushData
}

func (c *invitationPushClient) Push(_ context.Context, _ []string, data jpush.PushData) error {
	c.data = data
	return nil
}

func TestInvitationPushCarriesFixedDeadline(t *testing.T) {
	client := &invitationPushClient{}
	svc := &pushService{pushClient: client}
	event := &domain.FeedEvent{Type: "library", Source: "library", OccurredAt: 1, ExtendFields: map[string]string{"notification_type": "TEAM_INVITATION", "team_id": "1", "expires_at": "123"}}
	if err := svc.PushPreparedMSGWithCID(context.Background(), event, &PreparedPush{tokens: []string{"device"}}, "cid"); err != nil {
		t.Fatal(err)
	}
	if client.data.ExpiresAt == nil || client.data.ExpiresAt.Unix() != 123 || client.data.Cid != "cid" {
		t.Fatalf("data=%+v", client.data)
	}
}
