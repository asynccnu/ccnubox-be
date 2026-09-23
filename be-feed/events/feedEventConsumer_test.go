package events

import (
	"context"
	"errors"
	"testing"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

type testFeedService struct {
	service.FeedEventService
	calls int
	err   error
}

func (s *testFeedService) InsertEventList(context.Context, []domain.FeedEvent) error {
	s.calls++
	return s.err
}

type testFeedSession struct {
	sarama.ConsumerGroupSession
	marked int
}

func (s *testFeedSession) Context() context.Context { return context.Background() }
func (s *testFeedSession) MarkMessage(*sarama.ConsumerMessage, string) {
	s.marked++
}

type testFeedClaim struct {
	sarama.ConsumerGroupClaim
	messages chan *sarama.ConsumerMessage
}

func (c *testFeedClaim) Messages() <-chan *sarama.ConsumerMessage { return c.messages }

func TestFeedFailureNeverAcknowledgesOrSkips(t *testing.T) {
	const valid = `{"student_id":"student","type":"grade"}`
	for _, tc := range []struct {
		name      string
		payload   string
		err       error
		wantCalls int
	}{
		{name: "decode", payload: "invalid"},
		{name: "validation", payload: `{}`},
		{name: "storage", payload: valid, err: errors.New("unavailable"), wantCalls: 4},
		{name: "permanent_storage", payload: valid, err: &mysql.MySQLError{Number: 1406, Message: "data too long"}, wantCalls: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &testFeedService{err: tc.err}
			h := &feedEventKafkaHandler{consumer: &FeedEventConsumerHandler{
				feedService: svc,
				l:           zapx.NewZapLogger(zap.NewNop()),
			}}
			claim := &testFeedClaim{messages: make(chan *sarama.ConsumerMessage, 2)}
			claim.messages <- &sarama.ConsumerMessage{Topic: "feed_event", Offset: 1, Value: []byte(tc.payload)}
			claim.messages <- &sarama.ConsumerMessage{Topic: "feed_event", Offset: 2, Value: []byte(valid)}
			close(claim.messages)
			session := &testFeedSession{}
			if err := h.ConsumeClaim(session, claim); err == nil {
				t.Fatal("expected consumption failure")
			}
			if session.marked != 0 || svc.calls != tc.wantCalls || len(claim.messages) != 1 {
				t.Fatalf("marked=%d calls=%d remaining=%d", session.marked, svc.calls, len(claim.messages))
			}
		})
	}
}
