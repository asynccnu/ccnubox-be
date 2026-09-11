package grpc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type blockingFeedEventService struct {
	service.FeedEventService
	started chan struct{}
	release chan struct{}
}

func (s *blockingFeedEventService) PublicFeedEvent(context.Context, bool, domain.FeedEvent) (feedv1.PublishStatus, error) {
	close(s.started)
	<-s.release
	return feedv1.PublishStatus_ACCEPTED, nil
}

func TestValidatePublicFeedEventRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *feedv1.PublicFeedEventReq
		code codes.Code
	}{
		{name: "nil request", code: codes.InvalidArgument},
		{name: "nil event", req: &feedv1.PublicFeedEventReq{StudentId: "u"}, code: codes.InvalidArgument},
		{
			name: "library requires durable metadata",
			req:  &feedv1.PublicFeedEventReq{StudentId: "u", Event: &feedv1.FeedEvent{Type: feedv1.FeedEventType_LIBRARY}},
			code: codes.InvalidArgument,
		},
		{
			name: "library cannot contain credentials",
			req: &feedv1.PublicFeedEventReq{StudentId: "u", Event: &feedv1.FeedEvent{
				Type: feedv1.FeedEventType_LIBRARY, DedupeKey: "d", Source: "library", OccurredAt: 1,
				ExtendFields: map[string]string{"cas_token": "secret"},
			}},
			code: codes.InvalidArgument,
		},
		{
			name: "valid library event",
			req: &feedv1.PublicFeedEventReq{StudentId: "u", Event: &feedv1.FeedEvent{
				Type: feedv1.FeedEventType_LIBRARY, DedupeKey: "d", Source: "library", OccurredAt: 1,
				ExtendFields: map[string]string{"notification_type": "START_30"},
			}},
			code: codes.OK,
		},
		{
			name: "url at storage limit",
			req: &feedv1.PublicFeedEventReq{StudentId: "u", Event: &feedv1.FeedEvent{
				Type: feedv1.FeedEventType_GRADE, Url: strings.Repeat("a", 2047),
			}},
			code: codes.OK,
		},
		{
			name: "url exceeds storage limit",
			req: &feedv1.PublicFeedEventReq{StudentId: "u", Event: &feedv1.FeedEvent{
				Type: feedv1.FeedEventType_GRADE, Url: strings.Repeat("a", 2048),
			}},
			code: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePublicFeedEventRequest(tt.req)
			if got := status.Code(err); got != tt.code {
				t.Fatalf("code = %v, want %v (err=%v)", got, tt.code, err)
			}
		})
	}
}

func TestAllowListConversionPreservesOptionalLibrary(t *testing.T) {
	without := convAllowListFromGRPCToDomain(&feedv1.AllowList{StudentId: "u"})
	if without.Library != nil {
		t.Fatal("absent optional library field must remain nil")
	}
	disabled := false
	with := convAllowListFromGRPCToDomain(&feedv1.AllowList{StudentId: "u", Library: &disabled})
	if with.Library == nil || *with.Library {
		t.Fatal("explicit false library field was not preserved")
	}
}

func TestPublicFeedEventWaitsForBrokerAck(t *testing.T) {
	req := &feedv1.PublicFeedEventReq{StudentId: "u", Event: &feedv1.FeedEvent{Type: feedv1.FeedEventType_GRADE}}

	feedService := &blockingFeedEventService{started: make(chan struct{}), release: make(chan struct{})}
	server := &FeedServiceServer{feedEventService: feedService, l: zapx.NewZapLogger(zap.NewNop())}
	returned := make(chan error, 1)
	go func() {
		_, err := server.PublicFeedEvent(context.Background(), req)
		returned <- err
	}()
	select {
	case <-feedService.started:
	case <-time.After(time.Second):
		t.Fatal("publish did not reach producer")
	}
	select {
	case <-returned:
		t.Fatal("publish returned before broker acknowledgement")
	default:
	}
	close(feedService.release)
	select {
	case err := <-returned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("publish did not return after acknowledgement")
	}
}
