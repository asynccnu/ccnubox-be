package biz

import (
	"context"
)

// domain interfaces + usecase interface

type LibraryCrawler interface {
	GetSeatInfos(ctx context.Context, roomid string) ([]*Seat, error)
}

type LibraryUsecase interface {
	GetSeatFromCrawler(ctx context.Context, stuID string, RoomID string) ([]*Seat, error)
}
