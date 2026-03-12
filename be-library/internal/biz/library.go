package biz

import (
	"context"
)

type LibraryBiz interface {
	GetSeat(ctx context.Context, stuID string, RoomIDs []string) (map[string][]*Seat, error)
	ReserveSeat(ctx context.Context, stuID, devID, start, end string) (string, error)
	GetRecordByDate(ctx context.Context, stuID string, dateStr ...string) ([]*Record, error)
	GetCreditPoint(ctx context.Context, stuID string) (*CreditPoints, error)
	GetDiscussion(ctx context.Context, stuID, roomTypeID, venueID, date string) ([]*Discussion, error)
	SearchUser(ctx context.Context, stuID, studentID string) (*Search, error)
	ReserveDiscussion(ctx context.Context, stuID, devID, labID, kindID, title, start, end string, list []string) (string, error)
	CancelReserve(ctx context.Context, stuID, id string) (string, error)
	ReserveSeatRandomly(ctx context.Context, stuID, start, end string, roomIDs []string) (string, bool, error)
}

type LibraryCrawler interface {
	GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*Seat, error)
	ReserveSeat(ctx context.Context, stuID string, devid, start, end string) (string, error)
	GetTodayRecord(ctx context.Context, stuID string) ([]*Record, error)
	GetHistory(ctx context.Context, stuID string) ([]*Record, error)
	GetCreditPoint(ctx context.Context, stuID string) (*CreditPoints, error)
	GetDiscussion(ctx context.Context, stuID string, roomTypeId, venueId, date string) ([]*Discussion, error)
	SearchUser(ctx context.Context, stuID string, studentid string) (*Search, error)
	ReserveDiscussion(ctx context.Context, stuID string, devid, labid, kindid, title, start, end string, list []string) (string, error)
	CancelReserve(ctx context.Context, stuID string, id string) (string, error)
	GetFreeList(ctx context.Context, seatID string, stuID string) ([]*FreeTime, error)
}
