package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
	"go.uber.org/zap"
)

type testSession struct {
	sarama.ConsumerGroupSession
	ctx    context.Context
	marked int
}

func (s *testSession) Context() context.Context                    { return s.ctx }
func (s *testSession) MarkMessage(*sarama.ConsumerMessage, string) { s.marked++ }

type testClaim struct {
	sarama.ConsumerGroupClaim
	messages chan *sarama.ConsumerMessage
}

func (c testClaim) Messages() <-chan *sarama.ConsumerMessage { return c.messages }
func messages(values ...string) testClaim {
	ch := make(chan *sarama.ConsumerMessage, len(values))
	for _, value := range values {
		ch <- &sarama.ConsumerMessage{Topic: "test", Value: []byte(value)}
	}
	close(ch)
	return testClaim{messages: ch}
}

func TestUnacknowledgedMessageSurvivesFourFailures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	session := &testSession{ctx: ctx}
	calls := 0
	h := FuncConsumeHandler{log: zapx.NewZapLogger(zap.NewNop()), f: func(got context.Context, _ []byte, _ []byte) (bool, error) {
		calls++
		if got.Done() != ctx.Done() {
			t.Fatal("session cancellation was not propagated")
		}
		if calls == 5 {
			cancel()
		}
		return false, errors.New("next retry message was not published")
	}}
	claim := messages("failed", "later")
	_ = h.ConsumeClaim(session, claim)
	if calls != 5 || session.marked != 0 || len(claim.messages) != 1 {
		t.Fatalf("calls=%d marked=%d remaining=%d", calls, session.marked, len(claim.messages))
	}
}

type testProducer struct {
	sarama.SyncProducer
	send func(*sarama.ProducerMessage) error
}

func (p testProducer) SendMessage(m *sarama.ProducerMessage) (int32, int64, error) {
	return 0, 0, p.send(m)
}

func TestForwardingFailureDoesNotAcknowledge(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session := &testSession{ctx: ctx}
	calls := 0
	h := &DelaySendHandler{log: zapx.NewZapLogger(zap.NewNop()), sendGuard: saramax.NewSendGuard(1000, 1, nil), kp: testProducer{send: func(*sarama.ProducerMessage) error {
		calls++
		cancel()
		return errors.New("unknown broker error")
	}}}
	claim := messages("failed", "later")
	_ = h.ConsumeClaim(session, claim)
	if calls != 1 || session.marked != 0 || len(claim.messages) != 1 {
		t.Fatalf("calls=%d marked=%d remaining=%d", calls, session.marked, len(claim.messages))
	}
}

func TestCancelledSessionDoesNotRunBusiness(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	session := &testSession{ctx: ctx}
	h := FuncConsumeHandler{log: zapx.NewZapLogger(zap.NewNop()), f: func(context.Context, []byte, []byte) (bool, error) {
		t.Fatal("business ran after cancellation")
		return true, nil
	}}
	_ = h.ConsumeClaim(session, messages("queued"))
	if session.marked != 0 {
		t.Fatal("cancelled session acknowledged message")
	}
}

func TestConsumerRecoversClientCreationFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	c := NewConsumer(func() (sarama.Client, error) {
		calls++
		if calls == 2 {
			cancel()
		}
		return nil, errors.New("temporary startup failure")
	}, zapx.NewZapLogger(zap.NewNop()))
	defer c.Close()
	started := time.Now()
	if err := c.Consume(ctx, []string{"test"}, "group", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls != 2 || time.Since(started) < time.Second {
		t.Fatalf("calls=%d elapsed=%v", calls, time.Since(started))
	}
}

type lifecycleClient struct {
	sarama.Client
	closed *int
}

func (c lifecycleClient) Close() error { *c.closed++; return nil }

type lifecycleGroup struct {
	sarama.ConsumerGroup
	consume func(context.Context) error
	closed  *int
}

func (g lifecycleGroup) Consume(ctx context.Context, _ []string, _ sarama.ConsumerGroupHandler) error {
	return g.consume(ctx)
}
func (g lifecycleGroup) Close() error { *g.closed++; return nil }

func TestConsumerRecoversGroupCreationAndReleasesClients(t *testing.T) {
	created, closed, groups, closedGroups := 0, 0, 0, 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewConsumer(func() (sarama.Client, error) {
		created++
		return lifecycleClient{closed: &closed}, nil
	}, zapx.NewZapLogger(zap.NewNop()))
	defer c.Close()
	c.newGroup = func(string, sarama.Client) (sarama.ConsumerGroup, error) {
		groups++
		if groups == 1 {
			return nil, errors.New("coordinator temporarily unavailable")
		}
		return lifecycleGroup{closed: &closedGroups, consume: func(context.Context) error { cancel(); return nil }}, nil
	}
	if err := c.Consume(ctx, []string{"test"}, "group", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if created != 2 || closed != 2 || groups != 2 || closedGroups != 1 {
		t.Fatalf("clients=%d/%d groups=%d/%d", created, closed, groups, closedGroups)
	}
}

func TestConsumerRecoversUnknownSessionErrorWithoutRecreatingClient(t *testing.T) {
	created, closed, calls, closedGroups := 0, 0, 0, 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewConsumer(func() (sarama.Client, error) {
		created++
		return lifecycleClient{closed: &closed}, nil
	}, zapx.NewZapLogger(zap.NewNop()))
	defer c.Close()
	c.newGroup = func(string, sarama.Client) (sarama.ConsumerGroup, error) {
		return lifecycleGroup{closed: &closedGroups, consume: func(context.Context) error {
			calls++
			if calls == 2 {
				cancel()
			}
			return errors.New("unknown session error")
		}}, nil
	}
	if err := c.Consume(ctx, []string{"test"}, "group", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if created != 1 || closed != 1 || calls != 2 || closedGroups != 1 {
		t.Fatalf("clients=%d/%d calls=%d closed groups=%d", created, closed, calls, closedGroups)
	}
}

// 永久失败的消息阈值内保留位点并结束会话，达到阈值后跳过，后续消息继续处理。
func TestPermanentFailureSkippedAfterThreshold(t *testing.T) {
	l := zapx.NewZapLogger(zap.NewNop())
	h := FuncConsumeHandler{log: l, sk: saramax.NewSkipper(2, l), f: func(_ context.Context, _ []byte, value []byte) (bool, error) {
		if string(value) == "bad" {
			return false, saramax.Permanent(errors.New("invalid payload"))
		}
		return true, nil
	}}
	claimWith := func() testClaim {
		ch := make(chan *sarama.ConsumerMessage, 2)
		ch <- &sarama.ConsumerMessage{Topic: "test", Offset: 1, Value: []byte("bad")}
		ch <- &sarama.ConsumerMessage{Topic: "test", Offset: 2, Value: []byte("good")}
		close(ch)
		return testClaim{messages: ch}
	}

	// 阈值内：不确认位点，后面的消息也不处理。
	session := &testSession{ctx: context.Background()}
	first := claimWith()
	if err := h.ConsumeClaim(session, first); err != nil || session.marked != 0 || len(first.messages) != 1 {
		t.Fatalf("first round: err=%v marked=%d remaining=%d", err, session.marked, len(first.messages))
	}
	// 达到阈值：跳过坏消息，后续正常消息照常确认。
	second := claimWith()
	if err := h.ConsumeClaim(session, second); err != nil || session.marked != 2 || len(second.messages) != 0 {
		t.Fatalf("second round: err=%v marked=%d remaining=%d", err, session.marked, len(second.messages))
	}
}

func TestForwardingPermanentErrorSkipsAfterThresholdWithoutCoolingOthers(t *testing.T) {
	for _, failure := range []error{sarama.ErrMessageSizeTooLarge, sarama.ErrInvalidTopic} {
		l := zapx.NewZapLogger(zap.NewNop())
		guard := saramax.NewSendGuard(1000, 10, nil)
		h := &DelaySendHandler{log: l, sk: saramax.NewSkipper(2, l), sendGuard: guard, kp: testProducer{send: func(msg *sarama.ProducerMessage) error {
			value, _ := msg.Value.Encode()
			if string(value) == "bad" {
				return &sarama.ProducerError{Err: failure}
			}
			return nil
		}}}
		for round := range 2 {
			session := &testSession{ctx: context.Background()}
			claim := messages("bad", "good")
			if err := h.ConsumeClaim(session, claim); err != nil {
				t.Fatal(err)
			}
			want := 0
			if round == 1 {
				want = 2
			}
			if session.marked != want {
				t.Fatalf("round=%d marked=%d want=%d", round, session.marked, want)
			}
			if err := guard.Send(context.Background(), func() error { return nil }); err != nil {
				t.Fatalf("interactive send rejected: %v", err)
			}
		}
	}
}
