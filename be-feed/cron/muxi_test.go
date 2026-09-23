package cron

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

type scheduledMuxi struct {
	service.MuxiOfficialMSGService
	removed int
}

func (s *scheduledMuxi) GetToBePublicOfficialMSG(context.Context, bool) ([]domain.MuxiOfficialMSG, error) {
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
	err  error
	keys []string
}

func (s *scheduledFeed) PublicFeedEvent(_ context.Context, _ bool, event domain.FeedEvent) (feedv1.PublishStatus, error) {
	s.keys = append(s.keys, event.DedupeKey)
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
