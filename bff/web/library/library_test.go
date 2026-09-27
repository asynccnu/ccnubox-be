package library

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	libraryv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type fakeLibraryClient struct {
	libraryv1.LibraryServiceClient
	randomSeatResp *libraryv1.GetRandomSeatResponse
	randomSeatErr  error
	lastRandomReq  *libraryv1.GetRandomSeatRequest
	confirmResp    *libraryv1.ConfirmReservationResponse
	confirmErr     error
	lastConfirmReq *libraryv1.ConfirmReservationRequest
}

func (f *fakeLibraryClient) GetRandomSeat(ctx context.Context, in *libraryv1.GetRandomSeatRequest, opts ...grpc.CallOption) (*libraryv1.GetRandomSeatResponse, error) {
	f.lastRandomReq = in
	return f.randomSeatResp, f.randomSeatErr
}

func (f *fakeLibraryClient) ConfirmReservation(ctx context.Context, in *libraryv1.ConfirmReservationRequest, opts ...grpc.CallOption) (*libraryv1.ConfirmReservationResponse, error) {
	f.lastConfirmReq = in
	return f.confirmResp, f.confirmErr
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
