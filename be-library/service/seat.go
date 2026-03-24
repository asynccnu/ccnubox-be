package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	v1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var (
	ErrGetSeat  = errorx.FormatErrorFunc(v1.ErrorGetSeatError("获取座位失败"))
	ErrGetToken = errorx.FormatErrorFunc(v1.ErrorGetTokenError("获取token失败"))
)

type SeatService interface {
	GetSeat(ctx context.Context, req *v1.GetSeatRequest) (*v1.GetSeatResponse, error)
	GetSeatRecord(ctx context.Context, req *v1.GetSeatRecordRequest) (*v1.GetSeatRecordResponse, error)
}

type seatService struct {
	crawler    *crawler.Crawler
	userClient userv1.UserServiceClient
	l          logger.Logger
}

func NewSeatService(userClient userv1.UserServiceClient, libCrawler *crawler.Crawler, l logger.Logger) SeatService {
	return &seatService{
		crawler:    libCrawler,
		userClient: userClient,
		l:          l,
	}
}

func (s *seatService) GetSeat(ctx context.Context, req *v1.GetSeatRequest) (*v1.GetSeatResponse, error) {
	tokenResp, err := s.userClient.GetLibrarySeatToken(ctx, &userv1.GetLibraryTokenRequest{
		StudentId: req.StuId,
	})
	if err != nil {
		return nil, ErrGetToken(errorx.Errorf("get token failed, stuId: %s, err: %w", req.StuId, err))
	}

	roomSeatsMap, err := s.crawler.GetSeatInfos(ctx, tokenResp.Token, req.RoomIds)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("get seat infos failed, stuId: %s, err: %w", req.StuId, err))
	}

	// 尽量保持请求顺序，方便前端渲染
	resp := &v1.GetSeatResponse{
		RoomSeats: make([]*v1.RoomSeat, 0, len(req.RoomIds)),
	}
	for _, roomID := range req.RoomIds {
		seats := roomSeatsMap[roomID]
		resp.RoomSeats = append(resp.RoomSeats, &v1.RoomSeat{
			RoomId: roomID,
			Seats:  convertSeats(seats),
		})
	}

	return resp, nil
}

func (s *seatService) GetSeatRecord(ctx context.Context, req *v1.GetSeatRecordRequest) (*v1.GetSeatRecordResponse, error) {
	tokenResp, err := s.userClient.GetLibrarySeatToken(ctx, &userv1.GetLibraryTokenRequest{
		StudentId: req.StuId,
	})
	if err != nil {
		return nil, ErrGetToken(errorx.Errorf("get token failed, stuId: %s, err: %w", req.StuId, err))
	}

	var allRecords []*crawler.Record
	for _, dateType := range req.Date {
		switch dateType {
		case "today":
			records, err := s.crawler.GetTodayRecord(ctx, tokenResp.Token)
			if err != nil {
				return nil, ErrGetSeat(errorx.Errorf("get today record failed, stuId: %s, err: %w", req.StuId, err))
			}
			allRecords = append(allRecords, records...)
		case "history":
			records, err := s.crawler.GetHistory(ctx, tokenResp.Token)
			if err != nil {
				return nil, ErrGetSeat(errorx.Errorf("get history record failed, stuId: %s, err: %w", req.StuId, err))
			}
			allRecords = append(allRecords, records...)
		}
	}

	return &v1.GetSeatRecordResponse{
		Record: convertRecords(allRecords),
	}, nil
}

func convertSeats(src []*crawler.Seat) []*v1.Seat {
	if len(src) == 0 {
		return nil
	}
	result := make([]*v1.Seat, 0, len(src))
	for _, seat := range src {
		if seat == nil {
			continue
		}
		result = append(result, &v1.Seat{
			ID:        seat.ID,
			Label:     seat.Label,
			Name:      seat.Name,
			Status:    seat.Status,
			AfterFree: seat.AfterFree,
			FreeList:  convertFreeTimes(seat.FreeList),
		})
	}
	return result
}

func convertFreeTimes(src []*crawler.FreeTime) []*v1.FreeTime {
	if len(src) == 0 {
		return nil
	}
	result := make([]*v1.FreeTime, 0, len(src))
	for _, ts := range src {
		if ts == nil {
			continue
		}
		result = append(result, &v1.FreeTime{
			Start: ts.Start,
			End:   ts.End,
		})
	}
	return result
}

func convertRecords(src []*crawler.Record) []*v1.Record {
	if len(src) == 0 {
		return nil
	}
	result := make([]*v1.Record, 0, len(src))
	for _, r := range src {
		if r == nil {
			continue
		}
		result = append(result, &v1.Record{
			Id:        r.ID,
			RoomId:    r.RoomID,
			RoomName:  r.RoomName,
			BuildName: r.BuildName,
			FloorName: r.FloorName,
			SeatId:    r.SeatID,
			SeatLabel: r.SeatLabel,
			MakeBegin: r.MakeBegin.Format("2006-01-02 15:04"),
			MakeEnd:   r.MakeEnd.Format("2006-01-02 15:04"),
			MakeDate:  r.MakeDate.Format("2006-01-02"),
			Status:    r.Status,
			Message:   r.Message,
		})
	}
	return result
}
