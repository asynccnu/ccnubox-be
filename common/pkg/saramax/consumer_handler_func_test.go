package saramax

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type testSession struct {
	sarama.ConsumerGroupSession
	ctx    context.Context
	marked []int64
}

func (s *testSession) Context() context.Context { return s.ctx }
func (s *testSession) MarkMessage(msg *sarama.ConsumerMessage, _ string) {
	s.marked = append(s.marked, msg.Offset)
}

type testClaim struct {
	sarama.ConsumerGroupClaim
	messages chan *sarama.ConsumerMessage
	onRead   func()
}

func (c *testClaim) Messages() <-chan *sarama.ConsumerMessage {
	if c.onRead != nil {
		c.onRead()
	}
	return c.messages
}

func newTestClaim(values ...string) *testClaim {
	c := &testClaim{messages: make(chan *sarama.ConsumerMessage, len(values))}
	for i, value := range values {
		c.messages <- &sarama.ConsumerMessage{Topic: "test", Partition: 1, Offset: int64(i), Value: []byte(value)}
	}
	close(c.messages)
	return c
}

func TestHandlersDoNotSkipFailedOffsets(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, batch := range []bool{false, true} {
		for _, tc := range []struct {
			name       string
			values     []string
			batchSize  int
			failures   int
			wantCalls  int
			wantMarked []int64
			wantError  bool
		}{
			{name: "retry_success", values: []string{"1", "2"}, batchSize: 2, failures: 2, wantCalls: 3, wantMarked: []int64{0, 1}},
			{name: "retry_exhausted", values: []string{"1", "2", "3"}, batchSize: 2, failures: 4, wantCalls: 4, wantError: true},
			{name: "poison_first", values: []string{"invalid", "2"}, batchSize: 2, wantError: true},
			{name: "poison_after_prefix", values: []string{"1", "invalid", "3"}, batchSize: 3, wantCalls: 1, wantMarked: []int64{0}, wantError: true},
			{name: "failed_prefix_before_poison", values: []string{"1", "invalid", "3"}, batchSize: 3, failures: 4, wantCalls: 4, wantError: true},
			{name: "flush_on_close", values: []string{"1"}, batchSize: 3, wantCalls: 1, wantMarked: []int64{0}},
		} {
			name := "handler/" + tc.name
			if batch {
				name = "batch/" + tc.name
			}
			t.Run(name, func(t *testing.T) {
				l := zapx.NewZapLogger(zap.NewNop())
				cfg := &HandlerConfig{ConsumeNum: tc.batchSize, ConsumeTime: 1}
				session := &testSession{ctx: context.Background()}
				calls := 0
				fn := func(events []int) error {
					calls++
					if calls <= tc.failures {
						return failure
					}
					return nil
				}
				var handler sarama.ConsumerGroupHandler = NewHandler(l, cfg, fn)
				if batch {
					handler = NewBatchHandler(l, cfg, func(msgs []*sarama.ConsumerMessage, events []int) error {
						if len(msgs) != len(events) {
							t.Fatal("messages and events must match")
						}
						return fn(events)
					})
				}
				err := handler.ConsumeClaim(session, newTestClaim(tc.values...))
				if (err != nil) != tc.wantError || calls != tc.wantCalls || !reflect.DeepEqual(session.marked, tc.wantMarked) {
					t.Fatalf("err=%v calls=%d marked=%v; want error=%v calls=%d marked=%v", err, calls, session.marked, tc.wantError, tc.wantCalls, tc.wantMarked)
				}
			})
		}
	}
}

func TestHandlerCancellationDoesNotFlushOrMark(t *testing.T) {
	for _, mode := range []string{"before", "retry", "success"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "before" {
				cancel()
			}
			calls := 0
			h := NewContextHandler(zapx.NewZapLogger(zap.NewNop()), &HandlerConfig{ConsumeNum: 1, ConsumeTime: 1}, func(got context.Context, events []int) error {
				if got != ctx {
					t.Fatal("business handler did not receive session context")
				}
				calls++
				cancel()
				if mode == "retry" {
					return errors.New("failed")
				}
				return nil
			})
			session := &testSession{ctx: ctx}
			_ = h.ConsumeClaim(session, newTestClaim("1", "2"))
			wantCalls := 1
			if mode == "before" {
				wantCalls = 0
			}
			if calls != wantCalls || len(session.marked) != 0 {
				t.Fatalf("calls=%d marked=%v", calls, session.marked)
			}
		})
	}
}

type testConsumer struct {
	consume func() error
}

func (c testConsumer) Consume(context.Context, []string, sarama.ConsumerGroupHandler) error {
	return c.consume()
}

func TestRunConsumerBackoffAndCancellation(t *testing.T) {
	for _, consumeErr := range []error{nil, errors.New("broker unavailable")} {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		started := time.Now()
		RunConsumer(ctx, testConsumer{consume: func() error {
			calls++
			if calls == 2 {
				cancel()
			}
			return consumeErr
		}}, nil, nil, zapx.NewZapLogger(zap.NewNop()))
		cancel()
		if calls != 2 || time.Since(started) < time.Second {
			t.Fatalf("calls=%d elapsed=%v", calls, time.Since(started))
		}
	}
}

func TestHandlerCancellationLeavesBufferedBatchUnprocessed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reads := 0
	claim := &testClaim{messages: make(chan *sarama.ConsumerMessage, 1), onRead: func() {
		reads++
		if reads == 2 {
			cancel()
		}
	}}
	claim.messages <- &sarama.ConsumerMessage{Value: []byte("1")}
	calls := 0
	h := NewHandler(zapx.NewZapLogger(zap.NewNop()), &HandlerConfig{ConsumeNum: 2, ConsumeTime: 1}, func(events []int) error {
		calls++
		return nil
	})
	session := &testSession{ctx: ctx}
	_ = h.ConsumeClaim(session, claim)
	if calls != 0 || len(session.marked) != 0 || reads != 2 {
		t.Fatalf("calls=%d marked=%v reads=%d", calls, session.marked, reads)
	}
}

func TestHandlerTimerFlushUsesRetryAndSuccessOnlyAcknowledgement(t *testing.T) {
	for _, success := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		claim := &testClaim{messages: make(chan *sarama.ConsumerMessage, 1)}
		claim.messages <- &sarama.ConsumerMessage{Topic: "test", Offset: 10, Value: []byte("1")}
		calls := 0
		h := NewHandler(zapx.NewZapLogger(zap.NewNop()), &HandlerConfig{ConsumeNum: 2, ConsumeTime: 0}, func(events []int) error {
			calls++
			if success && calls == 2 {
				close(claim.messages)
				return nil
			}
			return errors.New("temporary failure")
		})
		session := &testSession{ctx: ctx}
		err := h.ConsumeClaim(session, claim)
		cancel()
		if success {
			if err != nil || calls != 2 || !reflect.DeepEqual(session.marked, []int64{10}) {
				t.Fatalf("successful timer flush: err=%v calls=%d marked=%v", err, calls, session.marked)
			}
		} else if err == nil || calls != 4 || len(session.marked) != 0 {
			t.Fatalf("failed timer flush: err=%v calls=%d marked=%v", err, calls, session.marked)
		}
	}
}

func TestBackoffIsBounded(t *testing.T) {
	var b Backoff
	for _, base := range []time.Duration{1, 2, 4, 8, 16, 30, 30, 30} {
		delay := b.Next()
		lower := base * time.Second
		if base == 30 {
			lower = 24 * time.Second
		}
		if delay < lower || delay > min(base*time.Second*5/4, 30*time.Second) {
			t.Fatalf("base=%v delay=%v", base, delay)
		}
	}
	b.Reset()
	if delay := b.Next(); delay < time.Second || delay > 1250*time.Millisecond {
		t.Fatalf("reset delay=%v", delay)
	}
}

func TestSendGuardRateCancellationAndClose(t *testing.T) {
	// 速率 50 次/秒、无突发额度：连续发送之间至少间隔 20 毫秒。
	guard := NewSendGuard(50, 1, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var previous time.Time
	for i := 0; i < 3; i++ {
		if err := guard.Send(ctx, func() error {
			now := time.Now()
			if !previous.IsZero() && now.Sub(previous) < 20*time.Millisecond {
				t.Fatal("send rate exceeded")
			}
			previous = now
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	cancel()
	if err := guard.Send(ctx, func() error { t.Fatal("sent after cancellation"); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled send: %v", err)
	}
	if err := guard.Close(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := guard.Send(context.Background(), func() error { t.Fatal("sent after close"); return nil }); !errors.Is(err, ErrProducerClosed) {
		t.Fatalf("closed send: %v", err)
	}
}

func TestSendGuardSharesFailureCooldown(t *testing.T) {
	guard := NewSendGuard(1000, 1, nil)
	failure := errors.New("Kafka unavailable")
	if err := guard.Send(context.Background(), func() error { return failure }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	results := make(chan error, 100)
	for i := 0; i < cap(results); i++ {
		go func() {
			results <- guard.Send(context.Background(), func() error {
				return errors.New("unexpected extra broker request")
			})
		}()
	}
	for i := 0; i < cap(results); i++ {
		if err := <-results; !errors.Is(err, ErrProducerCoolingDown) {
			t.Fatalf("request bypassed shared cooldown: %v", err)
		}
	}
	// 时间边界之外只放行一次探测，成功后才解除退避。
	guard.cooldown = time.Now().Add(-time.Second)
	if err := guard.Send(context.Background(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if guard.backoff.delay != 0 || !guard.cooldown.IsZero() {
		t.Fatal("successful probe did not reset cooldown")
	}
}

func TestSendGuardInflightSuccessDoesNotClearActiveCooldown(t *testing.T) {
	// 模拟：发送在途期间另一个发送失败设置了冷却。
	// 在途发送的成功不能清除仍在生效的冷却，否则持续故障期的退避会被并发成功反复清零。
	guard := NewSendGuard(1000, 10, nil)
	if err := guard.Send(context.Background(), func() error {
		guard.mu.Lock()
		guard.cooldown = time.Now().Add(time.Hour)
		guard.mu.Unlock()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	guard.mu.Lock()
	cooldown := guard.cooldown
	guard.mu.Unlock()
	if cooldown.IsZero() {
		t.Fatal("in-flight success cleared an active cooldown")
	}
	if err := guard.Send(context.Background(), func() error {
		t.Fatal("send passed while cooldown active")
		return nil
	}); !errors.Is(err, ErrProducerCoolingDown) {
		t.Fatalf("send during active cooldown: %v", err)
	}
}

func TestSendGuardAllowsBurstWithoutSpacing(t *testing.T) {
	// 突发额度 3：前三次立即发送，不额外等待；之后按 10 次/秒补充令牌。
	guard := NewSendGuard(10, 3, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := time.Now()
	for i := 0; i < 3; i++ {
		if err := guard.Send(ctx, func() error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(started); elapsed > 50*time.Millisecond {
		t.Fatalf("burst was delayed: %v", elapsed)
	}
	if err := guard.Send(ctx, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < 60*time.Millisecond {
		t.Fatalf("rate limit not applied after burst: %v", elapsed)
	}
}

func TestSendGuardWaitingSendDoesNotBlockOthers(t *testing.T) {
	// 令牌用尽后，等待令牌的发送不占用在途额度，不会阻塞其他调用方。
	guard := NewSendGuard(50, 1, nil)
	if err := guard.Send(context.Background(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- guard.Send(context.Background(), func() error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	// 第二次发送需要等令牌，但不应被第一次的在途发送卡住超过补令牌的时间。
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	ran := false
	if err := guard.Send(ctx, func() error { ran = true; return nil }); err != nil {
		t.Fatalf("waiting token send failed: %v", err)
	}
	if !ran {
		t.Fatal("send did not run within the token refill window")
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}

func TestConsumeEventsBlockedLogOnlyForLiveSessions(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	l := zapx.NewZapLogger(zap.New(core))
	blockedLogs := func() int {
		n := 0
		for _, e := range logs.All() {
			if strings.HasPrefix(e.Message, LogKeyPartitionBlocked) {
				n++
			}
		}
		return n
	}

	// 活会话：处理失败必须打 KAFKA_PARTITION_BLOCKED。
	h := NewHandler(l, &HandlerConfig{ConsumeNum: 1, ConsumeTime: 1, RetryAttempts: 1},
		func([]int) error { return errors.New("storage down") })
	if err := h.ConsumeClaim(&testSession{ctx: context.Background()}, newTestClaim("1")); err == nil {
		t.Fatal("expected error from live session")
	}
	if n := blockedLogs(); n != 1 {
		t.Fatalf("blocked logs = %d, want 1", n)
	}

	// 停机/再平衡：会话 ctx 已取消时的失败不是坏消息，不能误打 PARTITION_BLOCKED。
	ctx, cancel := context.WithCancel(context.Background())
	h2 := NewContextHandler(l, &HandlerConfig{ConsumeNum: 1, ConsumeTime: 1, RetryAttempts: 1},
		func(context.Context, []int) error {
			cancel()
			return errors.New("storage down")
		})
	_ = h2.ConsumeClaim(&testSession{ctx: ctx}, newTestClaim("1"))
	if n := blockedLogs(); n != 1 {
		t.Fatalf("blocked logs = %d, want 1 (no new entry for cancelled session)", n)
	}
}

func TestBatchHandlerPreservesOneSecondWindow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	claim := &testClaim{messages: make(chan *sarama.ConsumerMessage, 1)}
	claim.messages <- &sarama.ConsumerMessage{Value: []byte("1")}
	calls := 0
	h := NewBatchHandler(zapx.NewZapLogger(zap.NewNop()), &HandlerConfig{ConsumeNum: 10, ConsumeTime: 60}, func(_ []*sarama.ConsumerMessage, _ []int) error {
		calls++
		close(claim.messages)
		return nil
	})
	session := &testSession{ctx: ctx}
	if err := h.ConsumeClaim(session, claim); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(session.marked) != 1 {
		t.Fatalf("calls=%d marked=%v", calls, session.marked)
	}
}
