package biz

import (
	"context"
	"time"
)

type Record struct {
	ID        string
	RoomID    string
	RoomName  string
	BuildName string
	FloorName string
	SeatID    string
	SeatLabel string
	MakeBegin time.Time
	MakeEnd   time.Time
	MakeDate  time.Time
	Status    string
	Message   string
}

type RecordRepo interface {
	UpsertRecords(ctx context.Context, stuID string, list []*Record) error
	ListRecords(ctx context.Context, stuID string, date ...time.Time) ([]*Record, error)
	GetRecordUpdateTime(ctx context.Context, stuID string) (string, error)
}
