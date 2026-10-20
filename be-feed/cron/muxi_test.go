package cron

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

type scheduledMuxi struct {
	service.MuxiOfficialMSGService
	removed int
	msgs    []domain.MuxiOfficialMSG
}

func (s *scheduledMuxi) GetToBePublicOfficialMSG(context.Context, bool) ([]domain.MuxiOfficialMSG, error) {
	if s.msgs != nil {
		return s.msgs, nil
	}
	return []domain.MuxiOfficialMSG{{Id: "scheduled-id", Title: "标题", Content: "正文"}}, nil
}
func (s *scheduledMuxi) StopMuxiOfficialMSG(context.Context, string) error { s.removed++; return nil }

// invalidMuxi 返回一条永远无法入库的计划（正文超过字段上限），用于验证不会被永久阻塞。
type invalidMuxi struct {
	scheduledMuxi
}

func (s *invalidMuxi) GetToBePublicOfficialMSG(context.Context, bool) ([]domain.MuxiOfficialMSG, error) {
	return []domain.MuxiOfficialMSG{{Id: "invalid-id", Title: "标题", Content: strings.Repeat("长", domain.MaxFeedEventTextBytes)}}, nil
}

type scheduledFeed struct {
	service.FeedEventService
	err        error
	keys       []string
	users      []string
	broadcasts []bool
	urls       []string
}

func (s *scheduledFeed) PublicFeedEvent(_ context.Context, all bool, event domain.FeedEvent) (feedv1.PublishStatus, error) {
	s.keys = append(s.keys, event.DedupeKey)
	s.users = append(s.users, event.StudentId)
	s.broadcasts = append(s.broadcasts, all)
	s.urls = append(s.urls, event.Url)
	return feedv1.PublishStatus_ACCEPTED, s.err
}

func TestMuxiRemovesScheduleOnlyAfterPublishSucceeds(t *testing.T) {
	msgs := &scheduledMuxi{}
	feed := &scheduledFeed{err: errors.New("Kafka unavailable")}
	c := &MuxiController{ctx: context.Background(), muxi: msgs, feed: feed, l: zapx.NewZapLogger(zap.NewNop())}
	c.publicMuxiFeed()
	if msgs.removed != 0 {
		t.Fatal("failed broadcast removed schedule")
	}
	feed.err = nil
	c.publicMuxiFeed()
	if msgs.removed != 1 || len(feed.keys) != 2 || feed.keys[0] != "muxi:scheduled-id" || feed.keys[1] != feed.keys[0] {
		t.Fatalf("removed=%d keys=%v", msgs.removed, feed.keys)
	}
}

func TestMuxiDropsUnpublishableSchedule(t *testing.T) {
	msgs := &invalidMuxi{}
	feed := &scheduledFeed{}
	c := &MuxiController{ctx: context.Background(), muxi: msgs, feed: feed, l: zapx.NewZapLogger(zap.NewNop())}
	c.publicMuxiFeed()
	if msgs.removed != 1 || len(feed.keys) != 0 {
		t.Fatalf("removed=%d keys=%v", msgs.removed, feed.keys)
	}
}

func TestMuxiUsesOnlyFrozenRecipientsAndRetainsPlanOnFailure(t *testing.T) {
	msgs := &scheduledMuxi{msgs: []domain.MuxiOfficialMSG{{Id: "targeted", Title: "survey", Targeted: true, StudentIDs: []string{"a", "b"}, ExtendFields: domain.ExtendFields{"url": "https://example.com/survey"}}}}
	feed := &scheduledFeed{err: errors.New("Kafka unavailable")}
	c := &MuxiController{ctx: context.Background(), muxi: msgs, feed: feed, l: zapx.NewZapLogger(zap.NewNop())}
	c.publicMuxiFeed()
	if msgs.removed != 0 || len(feed.users) != 1 || feed.broadcasts[0] {
		t.Fatalf("failure must retain targeted schedule: %+v", feed)
	}
	feed.err = nil
	c.publicMuxiFeed()
	if msgs.removed != 1 || strings.Join(feed.users, ",") != "a,a,b" {
		t.Fatalf("recipients=%v removed=%d", feed.users, msgs.removed)
	}
	for i, key := range feed.keys {
		if key != "muxi:targeted" || feed.broadcasts[i] || feed.urls[i] != "https://example.com/survey" {
			t.Fatalf("wrong event: %+v", feed)
		}
	}
}
func TestMuxiEmptyTargetedPlanNeverBroadcasts(t *testing.T) {
	msgs := &scheduledMuxi{msgs: []domain.MuxiOfficialMSG{{Id: "empty", Targeted: true, Title: "empty"}}}
	feed := &scheduledFeed{}
	c := &MuxiController{ctx: context.Background(), muxi: msgs, feed: feed, l: zapx.NewZapLogger(zap.NewNop())}
	c.publicMuxiFeed()
	if len(feed.keys) != 0 || msgs.removed != 1 {
		t.Fatalf("empty targeted plan: calls=%d removed=%d", len(feed.keys), msgs.removed)
	}
}

func TestAllCurrentStudentsScheduleExcludesNewlyGraduatedUsers(t *testing.T) {
	msgs := &scheduledMuxi{msgs: []domain.MuxiOfficialMSG{{Id: "current-students", Title: "announcement", Targeted: true, WhiteListStrategy: "all", StudentIDs: []string{"20220001", "20230001"}}}}
	feed := &scheduledFeed{}
	now := time.Date(2026, 6, 30, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	c := &MuxiController{ctx: context.Background(), muxi: msgs, feed: feed, l: zapx.NewZapLogger(zap.NewNop()), now: func() time.Time { return now }}
	c.publicMuxiFeed()
	if msgs.removed != 1 || len(feed.users) != 1 || feed.users[0] != "20230001" || feed.broadcasts[0] {
		t.Fatalf("removed=%d users=%v broadcasts=%v", msgs.removed, feed.users, feed.broadcasts)
	}
}
