package service

import (
	"context"
	"errors"
	"testing"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/events/producer"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
)

type broadcastProducer struct {
	producer.Producer
	calls  []domain.FeedEvent
	failAt int
}

func (p *broadcastProducer) SendMessage(_ context.Context, _ string, event domain.FeedEvent) error {
	p.calls = append(p.calls, event)
	if len(p.calls) == p.failAt {
		return errors.New("Kafka unavailable")
	}
	return nil
}

type broadcastUsers struct{ dao.FeedUserConfigDAO }

func (broadcastUsers) GetStudentIdsByCursor(_ context.Context, lastID int64, _ int) ([]string, int64, error) {
	if lastID == 0 {
		return []string{"a", "b", "c"}, 3, nil
	}
	return nil, 3, nil
}

type broadcastEvents struct {
	dao.FeedEventDAO
	stored map[string]bool
}

func (d broadcastEvents) DedupeKeyExistsBatch(_ context.Context, studentIDs []string, key string) (map[string]bool, error) {
	existing := make(map[string]bool, len(studentIDs))
	for _, studentID := range studentIDs {
		if d.stored[studentID+":"+key] {
			existing[studentID] = true
		}
	}
	return existing, nil
}

func TestBroadcastStopsOnFailureAndReusesDedupeKey(t *testing.T) {
	p := &broadcastProducer{failAt: 2}
	stored := map[string]bool{}
	s := &feedEventService{feedProducer: p, feedUserConfigDAO: broadcastUsers{}, feedEventDAO: broadcastEvents{stored: stored},
		l: zapx.NewZapLogger(zap.NewNop())}
	event := domain.FeedEvent{Type: "muxi", DedupeKey: "announcement"}
	if _, err := s.PublicFeedEvent(context.Background(), true, event); err == nil {
		t.Fatal("broadcast hid partial failure")
	}
	if len(p.calls) != 2 || p.calls[1].StudentId != "b" {
		t.Fatalf("continued after failure: %+v", p.calls)
	}
	stored["a:announcement"] = true
	p.calls = nil
	p.failAt = 0
	if _, err := s.PublicFeedEvent(context.Background(), true, event); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 2 || p.calls[0].StudentId != "b" || p.calls[1].StudentId != "c" {
		t.Fatalf("resent completed prefix: %+v", p.calls)
	}
	for _, sent := range p.calls {
		if sent.DedupeKey != event.DedupeKey {
			t.Fatal("retry changed dedupe key")
		}
	}
}

func TestBroadcastRejectsMissingKeyBeforeSending(t *testing.T) {
	p := &broadcastProducer{}
	s := &feedEventService{feedProducer: p, l: zapx.NewZapLogger(zap.NewNop())}
	if _, err := s.PublicFeedEvent(context.Background(), true, domain.FeedEvent{Type: "muxi"}); err == nil {
		t.Fatal("broadcast accepted missing dedupe key")
	}
	if len(p.calls) != 0 {
		t.Fatal("sent invalid broadcast")
	}
}
