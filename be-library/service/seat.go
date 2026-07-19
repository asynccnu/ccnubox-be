package service

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/tool"
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
	ReserveSeat(ctx context.Context, req *v1.ReserveSeatRequest) (*v1.ReserveSeatResponse, error)
	GetSeatRecord(ctx context.Context, req *v1.GetSeatRecordRequest) (*v1.GetSeatRecordResponse, error)
	CancelReserve(ctx context.Context, req *v1.CancelReserveRequest) (*v1.CancelReserveResponse, error)
	ReserveSeatRandomly(ctx context.Context, req *v1.ReserveSeatRandomlyRequest) (*v1.ReserveSeatRandomlyResponse, error)
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
	if req == nil || len(req.RoomIds) == 0 {
		return nil, ErrGetSeat(errorx.New("room_ids must not be empty"))
	}
	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}

	roomSeatsMap, err := s.crawler.GetSeatInfos(ctx, token, req.RoomIds)
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

func (s *seatService) ReserveSeat(ctx context.Context, req *v1.ReserveSeatRequest) (*v1.ReserveSeatResponse, error) {
	if req == nil || req.DevId == "" || req.Start == "" || req.End == "" {
		return nil, ErrGetSeat(errorx.New("dev_id, start and end are required"))
	}
	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	message, err := s.crawler.ReserveSeat(ctx, token, req.DevId, req.Start, req.End)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("reserve seat failed, stuId: %s, err: %w", req.StuId, err))
	}
	return &v1.ReserveSeatResponse{Message: message}, nil
}

func (s *seatService) GetSeatRecord(ctx context.Context, req *v1.GetSeatRecordRequest) (*v1.GetSeatRecordResponse, error) {
	if req == nil || len(req.Date) == 0 {
		return nil, ErrGetSeat(errorx.New("date must not be empty"))
	}
	query, err := buildSeatRecordQuery(req.Date, time.Now())
	if err != nil {
		return nil, ErrGetSeat(err)
	}
	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}

	var allRecords []*crawler.Record
	if query.current {
		records, err := s.crawler.GetTodayRecord(ctx, token)
		if err != nil {
			return nil, ErrGetSeat(errorx.Errorf("get current seat records failed, stuId: %s, err: %w", req.StuId, err))
		}
		allRecords = appendSeatRecords(allRecords, records, query.dates, query.allCurrent)
	}
	if query.history {
		records, err := s.crawler.GetHistory(ctx, token)
		if err != nil {
			return nil, ErrGetSeat(errorx.Errorf("get history record failed, stuId: %s, err: %w", req.StuId, err))
		}
		allRecords = appendSeatRecords(allRecords, records, query.dates, query.allHistory)
	}
	allRecords = deduplicateSeatRecords(allRecords)

	return &v1.GetSeatRecordResponse{
		Record: convertRecords(allRecords),
	}, nil
}

type seatRecordQuery struct {
	dates      map[string]struct{}
	current    bool
	history    bool
	allCurrent bool
	allHistory bool
}

func buildSeatRecordQuery(values []string, now time.Time) (*seatRecordQuery, error) {
	loc, err := tool.GetLocation()
	if err != nil {
		return nil, errorx.Errorf("load library timezone: %w", err)
	}
	todayText := now.In(loc).Format("2006-01-02")
	today, err := time.ParseInLocation("2006-01-02", todayText, loc)
	if err != nil {
		return nil, errorx.Errorf("parse current date: %w", err)
	}
	query := &seatRecordQuery{dates: make(map[string]struct{}, len(values))}
	for _, rawValue := range values {
		value := strings.TrimSpace(rawValue)
		switch strings.ToLower(value) {
		case "today":
			query.current = true
			query.allCurrent = true
		case "history":
			query.history = true
			query.allHistory = true
		default:
			date, err := time.ParseInLocation("2006-1-2", value, loc)
			if err != nil {
				return nil, errorx.Errorf("invalid record date %q, expected YYYY-M-D", value)
			}
			query.dates[date.Format("2006-01-02")] = struct{}{}
			if date.Before(today) {
				query.history = true
			} else {
				query.current = true
			}
		}
	}
	if !query.current && !query.history {
		return nil, errorx.New("date must contain at least one valid value")
	}
	return query, nil
}

func appendSeatRecords(dst, records []*crawler.Record, dates map[string]struct{}, includeAll bool) []*crawler.Record {
	for _, record := range records {
		if record == nil {
			continue
		}
		if includeAll {
			dst = append(dst, record)
			continue
		}
		if _, ok := dates[record.MakeDate.Format("2006-01-02")]; ok {
			dst = append(dst, record)
		}
	}
	return dst
}

func deduplicateSeatRecords(records []*crawler.Record) []*crawler.Record {
	result := make([]*crawler.Record, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		key := record.ID
		if key == "" {
			key = strings.Join([]string{record.SeatID, record.MakeDate.Format("2006-01-02"), record.MakeBegin.Format("15:04"), record.MakeEnd.Format("15:04")}, "|")
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, record)
	}
	return result
}

func (s *seatService) CancelReserve(ctx context.Context, req *v1.CancelReserveRequest) (*v1.CancelReserveResponse, error) {
	if req == nil || req.Id == "" {
		return nil, ErrGetSeat(errorx.New("reservation id is required"))
	}
	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	message, err := s.crawler.CancelReserve(ctx, token, req.Id)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("cancel seat reservation failed, stuId: %s, err: %w", req.StuId, err))
	}
	return &v1.CancelReserveResponse{Message: message}, nil
}

func (s *seatService) ReserveSeatRandomly(ctx context.Context, req *v1.ReserveSeatRandomlyRequest) (*v1.ReserveSeatRandomlyResponse, error) {
	if req == nil || len(req.RoomId) == 0 || req.Start == "" || req.End == "" {
		return nil, ErrGetSeat(errorx.New("room_id, start and end are required"))
	}
	token, err := s.getSeatToken(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	roomSeats, err := s.crawler.GetSeatInfosForPeriod(ctx, token, req.RoomId, req.Start, req.End)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("find available seat failed, stuId: %s, err: %w", req.StuId, err))
	}
	var candidates []*crawler.Seat
	for _, roomID := range req.RoomId {
		candidates = append(candidates, roomSeats[roomID]...)
	}
	if len(candidates) == 0 {
		return nil, ErrGetSeat(errorx.New("no available seat for the requested time"))
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	seat := candidates[rng.Intn(len(candidates))]
	message, err := s.crawler.ReserveSeat(ctx, token, seat.ID, req.Start, req.End)
	if err != nil {
		return nil, ErrGetSeat(errorx.Errorf("reserve randomly selected seat failed, stuId: %s, err: %w", req.StuId, err))
	}
	return &v1.ReserveSeatRandomlyResponse{Message: message}, nil
}

func (s *seatService) getSeatToken(ctx context.Context, studentID string) (string, error) {
	if studentID == "" {
		return "", ErrGetToken(errorx.New("student id is required"))
	}
	tokenResp, err := s.userClient.GetLibrarySeatToken(ctx, &userv1.GetLibraryTokenRequest{StudentId: studentID})
	if err != nil {
		return "", ErrGetToken(errorx.Errorf("get token failed, stuId: %s, err: %w", studentID, err))
	}
	if tokenResp == nil || tokenResp.Token == "" {
		return "", ErrGetToken(errorx.Errorf("empty token returned, stuId: %s", studentID))
	}
	return tokenResp.Token, nil
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
