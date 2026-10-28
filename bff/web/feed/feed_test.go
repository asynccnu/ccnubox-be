package feed

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type feedClientStub struct {
	feedv1.FeedServiceClient
	changeReq        *feedv1.ChangeFeedAllowListReq
	readReq          *feedv1.ReadFeedEventReq
	allowList        *feedv1.AllowList
	changeErr        error
	readErr          error
	configStudentIDs []string
}

func (s *feedClientStub) ChangeFeedAllowList(
	_ context.Context,
	req *feedv1.ChangeFeedAllowListReq,
	_ ...grpc.CallOption,
) (*feedv1.ChangeFeedAllowListResp, error) {
	s.changeReq = req
	return &feedv1.ChangeFeedAllowListResp{}, s.changeErr
}

func (s *feedClientStub) FindOrCreateAllowList(
	_ context.Context,
	req *feedv1.FindOrCreateAllowListReq,
	_ ...grpc.CallOption,
) (*feedv1.FindOrCreateAllowListResp, error) {
	s.configStudentIDs = append(s.configStudentIDs, req.StudentId)
	return &feedv1.FindOrCreateAllowListResp{AllowList: s.allowList}, s.readErr
}

func (s *feedClientStub) ReadFeedEvent(
	_ context.Context,
	req *feedv1.ReadFeedEventReq,
	_ ...grpc.CallOption,
) (*feedv1.ReadFeedEventResp, error) {
	s.readReq = req
	return &feedv1.ReadFeedEventResp{}, nil
}

func boolPointer(value bool) *bool {
	return &value
}

func fullAllowListRequest(library *bool) ChangeFeedAllowListReq {
	return ChangeFeedAllowListReq{
		Grade:    boolPointer(true),
		Muxi:     boolPointer(false),
		Holiday:  boolPointer(true),
		Energy:   boolPointer(false),
		FeedBack: boolPointer(true),
		Library:  library,
	}
}

func TestChangeFeedAllowListPreservesAbsentLibrary(t *testing.T) {
	client := &feedClientStub{}
	handler := NewFeedHandler(client, nil, nil)

	_, err := handler.ChangeFeedAllowList(
		&gin.Context{},
		fullAllowListRequest(nil),
		ijwt.UserClaims{StudentId: "20260001"},
	)
	if err != nil {
		t.Fatalf("ChangeFeedAllowList() error = %v", err)
	}
	if client.changeReq == nil || client.changeReq.AllowList == nil {
		t.Fatal("ChangeFeedAllowList() did not call Feed with an allow list")
	}
	if client.changeReq.AllowList.StudentId != "20260001" {
		t.Fatalf("student ID = %q, want %q", client.changeReq.AllowList.StudentId, "20260001")
	}
	if client.changeReq.AllowList.Library != nil {
		t.Fatalf("library = %v, want nil for an old client request", *client.changeReq.AllowList.Library)
	}
}

func TestChangeFeedAllowListBindingAcceptsOldClientBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := &feedClientStub{}
	handler := NewFeedHandler(client, nil, nil)
	engine := gin.New()
	engine.POST("/allow-list", func(ctx *gin.Context) {
		ginx.SetClaims(ctx, ijwt.UserClaims{StudentId: "20260001"})
	}, ginx.WrapClaimsAndReq(handler.ChangeFeedAllowList))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/allow-list",
		bytes.NewBufferString(`{"grade":true,"muxi":false,"holiday":true,"energy":false,"feedback":true}`),
	)
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if client.changeReq == nil {
		t.Fatal("old client body without library was rejected before calling Feed")
	}
	if client.changeReq.AllowList.Library != nil {
		t.Fatal("old client body should forward an absent library field")
	}
}

func TestChangeFeedAllowListForwardsExplicitLibraryFalse(t *testing.T) {
	client := &feedClientStub{}
	handler := NewFeedHandler(client, nil, nil)

	_, err := handler.ChangeFeedAllowList(
		&gin.Context{},
		fullAllowListRequest(boolPointer(false)),
		ijwt.UserClaims{StudentId: "20260001"},
	)
	if err != nil {
		t.Fatalf("ChangeFeedAllowList() error = %v", err)
	}
	if client.changeReq.AllowList.Library == nil {
		t.Fatal("library = nil, want an explicitly present false value")
	}
	if *client.changeReq.AllowList.Library {
		t.Fatal("library = true, want false")
	}
}

func TestGetFeedAllowListIncludesLibrary(t *testing.T) {
	client := &feedClientStub{allowList: &feedv1.AllowList{Library: boolPointer(true)}}
	handler := NewFeedHandler(client, nil, nil)

	response, err := handler.GetFeedAllowList(
		&gin.Context{},
		ijwt.UserClaims{StudentId: "20260001"},
	)
	if err != nil {
		t.Fatalf("GetFeedAllowList() error = %v", err)
	}
	allowList, ok := response.Data.(GetFeedAllowListResp)
	if !ok {
		t.Fatalf("response data type = %T, want GetFeedAllowListResp", response.Data)
	}
	if !allowList.Library {
		t.Fatal("library = false, want true")
	}
}

func TestReadFeedEventInjectsStudentIDFromClaims(t *testing.T) {
	client := &feedClientStub{}
	handler := NewFeedHandler(client, nil, nil)

	_, err := handler.ReadFeedEvent(
		&gin.Context{},
		ReadFeedEventReq{FeedId: 42},
		ijwt.UserClaims{StudentId: "20260001"},
	)
	if err != nil {
		t.Fatalf("ReadFeedEvent() error = %v", err)
	}
	if client.readReq == nil {
		t.Fatal("ReadFeedEvent() did not call Feed")
	}
	if client.readReq.FeedId != 42 {
		t.Fatalf("feed ID = %d, want 42", client.readReq.FeedId)
	}
	if client.readReq.StudentId != "20260001" {
		t.Fatalf("student ID = %q, want %q", client.readReq.StudentId, "20260001")
	}
}

func TestAllSubscriptionsCanBeExplicitlyDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := &feedClientStub{}
	handler := NewFeedHandler(client, nil, nil)
	engine := gin.New()
	engine.POST("/allow-list", func(ctx *gin.Context) {
		ginx.SetClaims(ctx, ijwt.UserClaims{StudentId: "20260001"})
	}, ginx.WrapClaimsAndReq(handler.ChangeFeedAllowList))
	request := httptest.NewRequest(http.MethodPost, "/allow-list", bytes.NewBufferString(`{"grade":false,"muxi":false,"holiday":false,"energy":false,"feedback":false,"library":false}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(httptest.NewRecorder(), request)
	if client.changeReq == nil {
		t.Fatal("explicit all-false body rejected")
	}
	config := client.changeReq.AllowList
	if config.Grade || config.Muxi || config.Holiday || config.Energy || config.FeedBack || config.Library == nil || *config.Library {
		t.Fatalf("all-false request changed: %+v", config)
	}
}

func TestSubscriptionFailuresAreNotReportedAsSuccess(t *testing.T) {
	client := &feedClientStub{changeErr: errors.New("transaction rolled back"), readErr: errors.New("read failed")}
	handler := NewFeedHandler(client, nil, nil)
	if _, err := handler.ChangeFeedAllowList(&gin.Context{}, fullAllowListRequest(nil), ijwt.UserClaims{StudentId: "20260001"}); err == nil {
		t.Fatal("save error ignored")
	}
	if _, err := handler.GetFeedAllowList(&gin.Context{}, ijwt.UserClaims{StudentId: "20260001"}); err == nil {
		t.Fatal("readback error ignored")
	}
}

func TestSubscriptionReadUsesCurrentAccount(t *testing.T) {
	client := &feedClientStub{allowList: &feedv1.AllowList{FeedBack: true, Library: boolPointer(true)}}
	handler := NewFeedHandler(client, nil, nil)
	for _, sid := range []string{"20260001", "20260002"} {
		response, err := handler.GetFeedAllowList(&gin.Context{}, ijwt.UserClaims{StudentId: sid})
		if err != nil {
			t.Fatal(err)
		}
		config := response.Data.(GetFeedAllowListResp)
		if !config.FeedBack || !config.Library || config.Grade || config.Energy || config.Muxi || config.Holiday {
			t.Fatalf("six server switches not preserved: %+v", config)
		}
	}
	if len(client.configStudentIDs) != 2 || client.configStudentIDs[0] != "20260001" || client.configStudentIDs[1] != "20260002" {
		t.Fatalf("account not refreshed: %v", client.configStudentIDs)
	}
}
