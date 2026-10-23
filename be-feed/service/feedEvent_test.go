package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/events/producer"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
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
		prefix := fmt.Sprintf("%d", time.Now().Year()-1)
		return []string{prefix + "0001", prefix + "0002", prefix + "0003"}, 3, nil
	}
	return nil, 3, nil
}

type broadcastEvents struct {
	dao.FeedEventDAO
	stored map[string]bool
	err    error
}

func (d broadcastEvents) DedupeKeyExists(_ context.Context, studentID, key string) (bool, error) {
	return d.stored[studentID+":"+key], d.err
}

func (d broadcastEvents) DedupeKeyExistsBatch(_ context.Context, studentIDs []string, key string) (map[string]bool, error) {
	if d.err != nil {
		return nil, d.err
	}
	existing := make(map[string]bool, len(studentIDs))
	for _, studentID := range studentIDs {
		if d.stored[studentID+":"+key] {
			existing[studentID] = true
		}
	}
	return existing, nil
}

type libraryPublishUsers struct {
	dao.FeedUserConfigDAO
	enabled bool
}

func (d libraryPublishUsers) IsLibraryEnabled(context.Context, string) (bool, error) {
	return d.enabled, nil
}

func TestSinglePublishDedupeByRecipientAndKey(t *testing.T) {
	for _, eventType := range []string{"muxi", "grade", "holiday", "energy", "feedback", "library"} {
		t.Run(eventType, func(t *testing.T) {
			for _, tc := range []struct {
				name      string
				studentID string
				key       string
				duplicate bool
			}{
				{name: "stored message", studentID: "a", key: "announcement", duplicate: true},
				{name: "different recipient", studentID: "b", key: "announcement"},
				{name: "different key", studentID: "a", key: "another"},
				{name: "generated key", studentID: "a"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					p := &broadcastProducer{}
					s := &feedEventService{feedProducer: p,
						feedEventDAO:      broadcastEvents{stored: map[string]bool{"a:announcement": true}},
						feedUserConfigDAO: libraryPublishUsers{enabled: true}}
					event := domain.FeedEvent{StudentId: tc.studentID, Type: eventType, DedupeKey: tc.key}
					status, err := s.PublicFeedEvent(context.Background(), false, event)
					if err != nil {
						t.Fatal(err)
					}
					if tc.duplicate {
						if status != feedv1.PublishStatus_DUPLICATE || len(p.calls) != 0 {
							t.Fatalf("duplicate was sent: status=%v calls=%+v", status, p.calls)
						}
						return
					}
					if status != feedv1.PublishStatus_ACCEPTED || len(p.calls) != 1 {
						t.Fatalf("new message was not sent: status=%v calls=%+v", status, p.calls)
					}
					sent := p.calls[0]
					if sent.StudentId != tc.studentID || sent.DedupeKey == "" || (tc.key != "" && sent.DedupeKey != tc.key) {
						t.Fatalf("unexpected event: %+v", sent)
					}
				})
			}
		})
	}
}

func TestPublishStopsOnDedupeLookupFailure(t *testing.T) {
	for _, broadcast := range []bool{false, true} {
		t.Run(fmt.Sprintf("broadcast=%t", broadcast), func(t *testing.T) {
			p := &broadcastProducer{}
			s := &feedEventService{feedProducer: p, feedUserConfigDAO: broadcastUsers{},
				feedEventDAO: broadcastEvents{err: errors.New("database unavailable")}}
			event := domain.FeedEvent{StudentId: "a", Type: "muxi", DedupeKey: "announcement"}
			if _, err := s.PublicFeedEvent(context.Background(), broadcast, event); err == nil {
				t.Fatal("dedupe lookup failure was ignored")
			}
			if len(p.calls) != 0 {
				t.Fatalf("sent without dedupe check: %+v", p.calls)
			}
		})
	}
}

func TestLibraryPublishStillHonorsPreferenceBeforeDedupe(t *testing.T) {
	p := &broadcastProducer{}
	s := &feedEventService{feedProducer: p, feedUserConfigDAO: libraryPublishUsers{enabled: false}}
	status, err := s.PublicFeedEvent(context.Background(), false, domain.FeedEvent{StudentId: "a", Type: "library", DedupeKey: "reminder"})
	if err != nil || status != feedv1.PublishStatus_SUPPRESSED_BY_ALLOW_LIST || len(p.calls) != 0 {
		t.Fatalf("status=%v err=%v calls=%+v", status, err, p.calls)
	}
}

func TestTargetedPublishRetrySkipsStoredRecipients(t *testing.T) {
	p := &broadcastProducer{failAt: 2}
	stored := map[string]bool{}
	s := &feedEventService{feedProducer: p, feedEventDAO: broadcastEvents{stored: stored},
		l: zapx.NewZapLogger(zap.NewNop())}
	event := domain.FeedEvent{Type: "muxi", DedupeKey: "muxi:targeted"}
	publish := func() error {
		for _, studentID := range []string{"a", "b", "c"} {
			event.StudentId = studentID
			if _, err := s.PublicFeedEvent(context.Background(), false, event); err != nil {
				return err
			}
		}
		return nil
	}
	if err := publish(); err == nil {
		t.Fatal("targeted publish hid partial failure")
	}
	if len(p.calls) != 2 || p.calls[1].StudentId != "b" {
		t.Fatalf("continued after failure: %+v", p.calls)
	}
	// Kafka 消费者已将第一位收件人的消息落库，再用原 key 重跑整个白名单。
	stored["a:"+event.DedupeKey] = true
	p.calls = nil
	p.failAt = 0
	if err := publish(); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 2 || p.calls[0].StudentId != "b" || p.calls[1].StudentId != "c" {
		t.Fatalf("resent stored recipient: %+v", p.calls)
	}
	for _, sent := range p.calls {
		if sent.DedupeKey != event.DedupeKey {
			t.Fatalf("retry changed dedupe key: %+v", sent)
		}
	}
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
	if len(p.calls) != 2 || p.calls[1].StudentId != fmt.Sprintf("%d0002", time.Now().Year()-1) {
		t.Fatalf("continued after failure: %+v", p.calls)
	}
	stored[fmt.Sprintf("%d0001:announcement", time.Now().Year()-1)] = true
	p.calls = nil
	p.failAt = 0
	if _, err := s.PublicFeedEvent(context.Background(), true, event); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 2 || p.calls[0].StudentId != fmt.Sprintf("%d0002", time.Now().Year()-1) || p.calls[1].StudentId != fmt.Sprintf("%d0003", time.Now().Year()-1) {
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

func TestLegacyOfficialBroadcastSkipsGraduatedAndMalformedIDs(t *testing.T) {
	p := &broadcastProducer{}
	current := fmt.Sprintf("%d0001", time.Now().Year()-1)
	users := &officialUsersStub{batches: [][]string{{"19990001", "bad"}, {current, "20990001"}}}
	s := &feedEventService{feedProducer: p, feedUserConfigDAO: users, feedEventDAO: broadcastEvents{stored: map[string]bool{}}, l: zapx.NewZapLogger(zap.NewNop())}
	if _, err := s.PublicFeedEvent(context.Background(), true, domain.FeedEvent{Type: "muxi", DedupeKey: "legacy"}); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || p.calls[0].StudentId != current || len(users.cursors) != 3 {
		t.Fatalf("events=%+v cursors=%v", p.calls, users.cursors)
	}
}
