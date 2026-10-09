package library

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type invitationTestClient struct {
	libraryv1.LibraryServiceClient
	req *libraryv1.NotifyTeamInvitationRequest
	err error
}

func (c *invitationTestClient) NotifyTeamInvitation(_ context.Context, req *libraryv1.NotifyTeamInvitationRequest, _ ...grpc.CallOption) (*libraryv1.NotifyTeamInvitationResponse, error) {
	c.req = req
	return &libraryv1.NotifyTeamInvitationResponse{}, c.err
}

func TestNotifyInvitationBinding(t *testing.T) {
	for _, tc := range []struct {
		name, body     string
		want, wantCode int
	}{
		{"valid", `{"team_id":"2102744440918429696","student_ids":["20260001","研究生:1","20260001"]}`, 200, 0},
		{"unknown", `{"team_id":"1","student_ids":["A"],"operator_student_id":"other"}`, 400, 41514},
		{"number_team", `{"team_id":1,"student_ids":["A"]}`, 400, 41514},
		{"number_student", `{"team_id":"1","student_ids":[1]}`, 400, 41514},
		{"empty", `{"team_id":"1","student_ids":[]}`, 400, 41514},
		{"self", `{"team_id":"1","student_ids":["operator"]}`, 400, 41514},
		{"whitespace", `{"team_id":"1","student_ids":[" A"]}`, 400, 41514},
		{"trailing", `{"team_id":"1","student_ids":["A"]} {}`, 400, 41514},
		{"unknown_title", `{"team_id":"1","student_ids":["A"],"title":"hi"}`, 400, 41514},
		{"oversized", strings.Repeat(" ", 8193), 413, 41517},
		{"raw_count", `{"team_id":"1","student_ids":[` + strings.Repeat(`"A",`, 20) + `"A"]}`, 400, 41514},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &invitationTestClient{}
			handler := NewLibraryHandler(client, nil)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			resp, err := handler.NotifyTeamInvitation(ctx, ijwt.UserClaims{StudentId: "operator"})
			got, gotCode := 200, resp.Code
			if err != nil {
				var custom *b_errorx.CustomError
				if !errors.As(err, &custom) {
					t.Fatal(err)
				}
				got, gotCode = custom.HttpCode, custom.Code
			}
			if got != tc.want || gotCode != tc.wantCode {
				t.Fatalf("status=%d code=%d want status=%d code=%d err=%v", got, gotCode, tc.want, tc.wantCode, err)
			}
			if got == 200 {
				if client.req.OperatorStudentId != "operator" || client.req.TeamId != "2102744440918429696" || len(client.req.StudentIds) != 3 || resp.Data.(NotifyTeamInvitationResponse).Status != "accepted" {
					t.Fatalf("request=%v response=%v", client.req, resp)
				}
			} else if client.req != nil {
				t.Fatal("invalid request reached RPC")
			}
		})
	}
}

func TestNotifyInvitationErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		code           codes.Code
		want, wantCode int
	}{
		{codes.InvalidArgument, 400, 41514},
		{codes.PermissionDenied, 403, 41515},
		{codes.FailedPrecondition, 409, 41516},
		{codes.ResourceExhausted, 429, 41518},
		{codes.Unavailable, 503, 51521},
		{codes.Unauthenticated, 503, 51521},
		{codes.DeadlineExceeded, 503, 51521},
		{codes.Internal, 500, 51520},
	} {
		client := &invitationTestClient{err: status.Error(tc.code, "sensitive upstream body")}
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"team_id":"1","student_ids":["A"]}`))
		_, err := NewLibraryHandler(client, nil).NotifyTeamInvitation(ctx, ijwt.UserClaims{StudentId: "operator"})
		var custom *b_errorx.CustomError
		if !errors.As(err, &custom) || custom.HttpCode != tc.want || custom.Code != tc.wantCode || strings.Contains(err.Error(), "sensitive") {
			t.Fatalf("code=%v err=%v", tc.code, err)
		}
		if tc.want == 429 && recorder.Header().Get("Retry-After") != "60" {
			t.Fatal("missing retry hint")
		}
	}
}

func TestNotifyInvitationRouteRequiresAuthentication(t *testing.T) {
	client := &invitationTestClient{}
	engine := gin.New()
	NewLibraryHandler(client, nil).RegisterRoutes(engine.Group("/api/v1"), func(ctx *gin.Context) { ctx.AbortWithStatus(http.StatusUnauthorized) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/library/notify_team_invitation", strings.NewReader(`{}`)))
	if recorder.Code != 401 || client.req != nil {
		t.Fatalf("status=%d req=%v", recorder.Code, client.req)
	}
	// 有效身份只从中间件 claims 传入。
	engine = gin.New()
	NewLibraryHandler(client, nil).RegisterRoutes(engine.Group("/api/v1"), func(ctx *gin.Context) { ginx.SetClaims(ctx, ijwt.UserClaims{StudentId: "operator"}) })
	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/library/notify_team_invitation", strings.NewReader(`{"team_id":"1","student_ids":["A"]}`)))
	if client.req == nil || client.req.OperatorStudentId != "operator" {
		t.Fatalf("req=%v", client.req)
	}
}

type fakeLibraryClient struct {
	libraryv1.LibraryServiceClient
	randomSeatResp      *libraryv1.GetRandomSeatResponse
	randomSeatErr       error
	lastRandomReq       *libraryv1.GetRandomSeatRequest
	confirmResp         *libraryv1.ConfirmReservationResponse
	confirmErr          error
	lastConfirmReq      *libraryv1.ConfirmReservationRequest
	smartPlansResp      *libraryv1.GetSmartSeatPlansResponse
	smartPlansErr       error
	lastSmartPlansReq   *libraryv1.GetSmartSeatPlansRequest
	reserveSmartResp    *libraryv1.ReserveSmartSeatPlanResponse
	reserveSmartErr     error
	lastReserveSmartReq *libraryv1.ReserveSmartSeatPlanRequest
	cancelSmartResp     *libraryv1.CancelSmartSeatPlanResponse
	cancelSmartErr      error
	lastCancelSmartReq  *libraryv1.CancelSmartSeatPlanRequest
}

func (f *fakeLibraryClient) GetRandomSeat(ctx context.Context, in *libraryv1.GetRandomSeatRequest, opts ...grpc.CallOption) (*libraryv1.GetRandomSeatResponse, error) {
	f.lastRandomReq = in
	return f.randomSeatResp, f.randomSeatErr
}

func (f *fakeLibraryClient) ConfirmReservation(ctx context.Context, in *libraryv1.ConfirmReservationRequest, opts ...grpc.CallOption) (*libraryv1.ConfirmReservationResponse, error) {
	f.lastConfirmReq = in
	return f.confirmResp, f.confirmErr
}

func (f *fakeLibraryClient) GetSmartSeatPlans(ctx context.Context, in *libraryv1.GetSmartSeatPlansRequest, opts ...grpc.CallOption) (*libraryv1.GetSmartSeatPlansResponse, error) {
	f.lastSmartPlansReq = in
	return f.smartPlansResp, f.smartPlansErr
}

func (f *fakeLibraryClient) ReserveSmartSeatPlan(ctx context.Context, in *libraryv1.ReserveSmartSeatPlanRequest, opts ...grpc.CallOption) (*libraryv1.ReserveSmartSeatPlanResponse, error) {
	f.lastReserveSmartReq = in
	return f.reserveSmartResp, f.reserveSmartErr
}

func (f *fakeLibraryClient) CancelSmartSeatPlan(ctx context.Context, in *libraryv1.CancelSmartSeatPlanRequest, opts ...grpc.CallOption) (*libraryv1.CancelSmartSeatPlanResponse, error) {
	f.lastCancelSmartReq = in
	return f.cancelSmartResp, f.cancelSmartErr
}

func newLibraryTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	return ctx
}

func assertCustomErrorCode(t *testing.T, err error, wantCode, wantStatus int) {
	t.Helper()
	var got *b_errorx.CustomError
	if !errors.As(err, &got) {
		t.Fatalf("error %v does not contain a CustomError", err)
	}
	if got.Code != wantCode {
		t.Errorf("code = %d, want %d", got.Code, wantCode)
	}
	if got.HttpCode != wantStatus {
		t.Errorf("HTTP status = %d, want %d", got.HttpCode, wantStatus)
	}
}

func TestGetRandomSeat(t *testing.T) {
	claims := ijwt.UserClaims{StudentId: "2025211366"}

	t.Run("success", func(t *testing.T) {
		client := &fakeLibraryClient{
			randomSeatResp: &libraryv1.GetRandomSeatResponse{
				RoomId: "room-1",
				Seat: &libraryv1.Seat{
					ID:        "s1",
					Label:     "A区 K3",
					Name:      "K3",
					Status:    "1",
					AfterFree: true,
				},
			},
		}
		handler := NewLibraryHandler(client, nil)
		resp, err := handler.GetRandomSeat(newLibraryTestContext(), GetRandomSeatRequest{
			RoomIDs:        []string{"room-1"},
			Date:           "2026-09-10",
			Start:          "14:30",
			End:            "16:30",
			ExcludeSeatIDs: []string{"s0"},
		}, claims)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := resp.Data.(GetRandomSeatResponse)
		if !ok {
			t.Fatalf("unexpected response data type %T", resp.Data)
		}
		if data.RoomID != "room-1" || data.Seat.ID != "s1" || data.Seat.Label != "A区 K3" || !data.Seat.AfterFree {
			t.Fatalf("unexpected response data: %+v", data)
		}

		req := client.lastRandomReq
		if req == nil {
			t.Fatal("request was not forwarded")
		}
		if req.StuId != claims.StudentId || req.Date != "2026-09-10" || req.Start != "14:30" || req.End != "16:30" {
			t.Fatalf("unexpected request: %+v", req)
		}
		if len(req.RoomIds) != 1 || req.RoomIds[0] != "room-1" {
			t.Fatalf("unexpected room ids: %v", req.RoomIds)
		}
		if len(req.ExcludeSeatIds) != 1 || req.ExcludeSeatIds[0] != "s0" {
			t.Fatalf("unexpected exclude seat ids: %v", req.ExcludeSeatIds)
		}
	})

	t.Run("no available seat", func(t *testing.T) {
		client := &fakeLibraryClient{randomSeatErr: libraryv1.ErrorNoAvailableSeatError("该时段无可用座位")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.GetRandomSeat(newLibraryTestContext(), GetRandomSeatRequest{
			RoomIDs: []string{"room-1"}, Date: "2026-09-10", Start: "14:30", End: "16:30",
		}, claims)
		assertCustomErrorCode(t, err, errs.NO_AVAILABLE_SEAT_ERROR_CODE, 500)
	})

	t.Run("unexpected error", func(t *testing.T) {
		client := &fakeLibraryClient{randomSeatErr: errors.New("upstream unavailable")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.GetRandomSeat(newLibraryTestContext(), GetRandomSeatRequest{
			RoomIDs: []string{"room-1"}, Date: "2026-09-10", Start: "14:30", End: "16:30",
		}, claims)
		assertCustomErrorCode(t, err, errs.GET_RANDOM_SEAT_ERROR_CODE, 500)
	})
}

func TestConfirmReservation(t *testing.T) {
	claims := ijwt.UserClaims{StudentId: "2025211366"}

	t.Run("success", func(t *testing.T) {
		client := &fakeLibraryClient{confirmResp: &libraryv1.ConfirmReservationResponse{Message: "预约成功"}}
		handler := NewLibraryHandler(client, nil)
		resp, err := handler.ConfirmReservation(newLibraryTestContext(), ConfirmReservationRequest{
			DevID: "s1", Date: "2026-09-10", Start: "14:30", End: "16:30",
		}, claims)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Msg != "预约成功" {
			t.Fatalf("message = %q, want %q", resp.Msg, "预约成功")
		}

		req := client.lastConfirmReq
		if req == nil {
			t.Fatal("request was not forwarded")
		}
		if req.StuId != claims.StudentId || req.DevId != "s1" || req.Date != "2026-09-10" || req.Start != "14:30" || req.End != "16:30" {
			t.Fatalf("unexpected request: %+v", req)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		client := &fakeLibraryClient{confirmErr: errors.New("seat already taken")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.ConfirmReservation(newLibraryTestContext(), ConfirmReservationRequest{
			DevID: "s1", Date: "2026-09-10", Start: "14:30", End: "16:30",
		}, claims)
		assertCustomErrorCode(t, err, errs.CONFIRM_RESERVATION_ERROR_CODE, 500)
	})
}

func TestGetSmartSeatPlans(t *testing.T) {
	claims := ijwt.UserClaims{StudentId: "2025211366"}

	t.Run("success", func(t *testing.T) {
		client := &fakeLibraryClient{
			smartPlansResp: &libraryv1.GetSmartSeatPlansResponse{
				Plans: []*libraryv1.SmartSeatPlan{
					{
						SegmentCount: 1,
						Segments: []*libraryv1.SmartSeatSegment{
							{SeatId: "s1", SeatLabel: "A区 K3", SeatName: "K3", RoomId: "room-1", Start: "09:00", End: "12:00"},
						},
					},
					{
						SegmentCount: 2,
						Segments: []*libraryv1.SmartSeatSegment{
							{SeatId: "s2", SeatLabel: "A区 K4", SeatName: "K4", RoomId: "room-1", Start: "09:00", End: "10:00"},
							{SeatId: "s3", SeatLabel: "B区 K1", SeatName: "K1", RoomId: "room-2", Start: "10:00", End: "12:00"},
						},
					},
				},
			},
		}
		handler := NewLibraryHandler(client, nil)
		resp, err := handler.GetSmartSeatPlans(newLibraryTestContext(), GetSmartSeatRequest{
			RoomIDs: []string{"room-1", "room-2"}, Date: "2026-09-10", Start: "09:00", End: "12:00",
		}, claims)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := resp.Data.(GetSmartSeatResponse)
		if !ok {
			t.Fatalf("unexpected response data type %T", resp.Data)
		}
		if len(data.Plans) != 2 {
			t.Fatalf("got %d plans, want 2", len(data.Plans))
		}
		first := data.Plans[0]
		if first.SegmentCount != 1 || len(first.Segments) != 1 {
			t.Fatalf("unexpected first plan: %+v", first)
		}
		if first.Segments[0].SeatID != "s1" || first.Segments[0].SeatLabel != "A区 K3" || first.Segments[0].RoomID != "room-1" || first.Segments[0].Start != "09:00" || first.Segments[0].End != "12:00" {
			t.Fatalf("unexpected first segment: %+v", first.Segments[0])
		}
		second := data.Plans[1]
		if second.SegmentCount != 2 || len(second.Segments) != 2 {
			t.Fatalf("unexpected second plan: %+v", second)
		}
		if second.Segments[1].SeatID != "s3" || second.Segments[1].RoomID != "room-2" || second.Segments[1].Start != "10:00" || second.Segments[1].End != "12:00" {
			t.Fatalf("unexpected segment: %+v", second.Segments[1])
		}

		req := client.lastSmartPlansReq
		if req == nil {
			t.Fatal("request was not forwarded")
		}
		if req.StuId != claims.StudentId || req.Date != "2026-09-10" || req.Start != "09:00" || req.End != "12:00" {
			t.Fatalf("unexpected request: %+v", req)
		}
		if len(req.RoomIds) != 2 || req.RoomIds[0] != "room-1" || req.RoomIds[1] != "room-2" {
			t.Fatalf("unexpected room ids: %v", req.RoomIds)
		}
	})

	t.Run("no available plan", func(t *testing.T) {
		client := &fakeLibraryClient{smartPlansErr: libraryv1.ErrorNoAvailableSeatError("该时段无可用座位")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.GetSmartSeatPlans(newLibraryTestContext(), GetSmartSeatRequest{
			RoomIDs: []string{"room-1"}, Date: "2026-09-10", Start: "09:00", End: "12:00",
		}, claims)
		assertCustomErrorCode(t, err, errs.NO_AVAILABLE_SEAT_ERROR_CODE, 500)
	})

	t.Run("unexpected error", func(t *testing.T) {
		client := &fakeLibraryClient{smartPlansErr: errors.New("upstream unavailable")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.GetSmartSeatPlans(newLibraryTestContext(), GetSmartSeatRequest{
			RoomIDs: []string{"room-1"}, Date: "2026-09-10", Start: "09:00", End: "12:00",
		}, claims)
		assertCustomErrorCode(t, err, errs.GET_SMART_SEAT_ERROR_CODE, 500)
	})
}

func TestReserveSmartSeatPlan(t *testing.T) {
	claims := ijwt.UserClaims{StudentId: "2025211366"}
	segments := []SmartSeatSegment{
		{SeatID: "s1", SeatLabel: "A区 K3", SeatName: "K3", RoomID: "room-1", Start: "09:00", End: "10:00"},
		{SeatID: "s2", SeatLabel: "B区 K1", SeatName: "K1", RoomID: "room-2", Start: "10:00", End: "12:00"},
	}

	t.Run("success", func(t *testing.T) {
		client := &fakeLibraryClient{reserveSmartResp: &libraryv1.ReserveSmartSeatPlanResponse{Message: "预约成功，共 2 个座位"}}
		handler := NewLibraryHandler(client, nil)
		resp, err := handler.ReserveSmartSeatPlan(newLibraryTestContext(), ReserveSmartSeatRequest{
			Date: "2026-09-10", Segments: segments,
		}, claims)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Msg != "预约成功，共 2 个座位" {
			t.Fatalf("message = %q", resp.Msg)
		}

		req := client.lastReserveSmartReq
		if req == nil {
			t.Fatal("request was not forwarded")
		}
		if req.StuId != claims.StudentId || req.Date != "2026-09-10" || len(req.Segments) != 2 {
			t.Fatalf("unexpected request: %+v", req)
		}
		if req.Segments[0].SeatId != "s1" || req.Segments[0].RoomId != "room-1" || req.Segments[0].Start != "09:00" || req.Segments[0].End != "10:00" {
			t.Fatalf("unexpected first segment: %+v", req.Segments[0])
		}
		if req.Segments[1].SeatId != "s2" || req.Segments[1].RoomId != "room-2" || req.Segments[1].Start != "10:00" || req.Segments[1].End != "12:00" {
			t.Fatalf("unexpected second segment: %+v", req.Segments[1])
		}
	})

	t.Run("no available seat", func(t *testing.T) {
		client := &fakeLibraryClient{reserveSmartErr: libraryv1.ErrorNoAvailableSeatError("座位已被占用")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.ReserveSmartSeatPlan(newLibraryTestContext(), ReserveSmartSeatRequest{
			Date: "2026-09-10", Segments: segments,
		}, claims)
		assertCustomErrorCode(t, err, errs.NO_AVAILABLE_SEAT_ERROR_CODE, 500)
	})

	t.Run("unexpected error", func(t *testing.T) {
		client := &fakeLibraryClient{reserveSmartErr: errors.New("upstream unavailable")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.ReserveSmartSeatPlan(newLibraryTestContext(), ReserveSmartSeatRequest{
			Date: "2026-09-10", Segments: segments,
		}, claims)
		assertCustomErrorCode(t, err, errs.RESERVE_SMART_SEAT_ERROR_CODE, 500)
	})
}

func TestCancelSmartSeatPlan(t *testing.T) {
	claims := ijwt.UserClaims{StudentId: "2025211366"}

	t.Run("success", func(t *testing.T) {
		client := &fakeLibraryClient{cancelSmartResp: &libraryv1.CancelSmartSeatPlanResponse{Message: "已取消 2 个座位"}}
		handler := NewLibraryHandler(client, nil)
		resp, err := handler.CancelSmartSeatPlan(newLibraryTestContext(), CancelSmartSeatRequest{Date: "2026-09-10"}, claims)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Msg != "已取消 2 个座位" {
			t.Fatalf("message = %q", resp.Msg)
		}

		req := client.lastCancelSmartReq
		if req == nil {
			t.Fatal("request was not forwarded")
		}
		if req.StuId != claims.StudentId || req.Date != "2026-09-10" {
			t.Fatalf("unexpected request: %+v", req)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		client := &fakeLibraryClient{cancelSmartErr: errors.New("cancel failed")}
		handler := NewLibraryHandler(client, nil)
		_, err := handler.CancelSmartSeatPlan(newLibraryTestContext(), CancelSmartSeatRequest{Date: "2026-09-10"}, claims)
		assertCustomErrorCode(t, err, errs.CANCEL_SMART_SEAT_ERROR_CODE, 500)
	})
}
