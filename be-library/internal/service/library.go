package service

import (
	"context"

	pb "github.com/asynccnu/ccnubox-be/be-api/gen/proto/library/v1"
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
)

type LibraryService struct {
	pb.UnimplementedLibraryServer
	use biz.LibraryUsecase
	log *log.Helper
}

func NewLibraryService(use biz.LibraryUsecase, logger log.Logger) *LibraryService {
	return &LibraryService{
		use: use,
		log: log.NewHelper(logger),
	}
}

func (ls *LibraryService) GetSeat(ctx context.Context, req *pb.GetSeatRequest) (*pb.GetSeatResponse, error) {
	// 调用 usecase 层方法，爬取(或获取数据)
	seats, err := ls.use.GetSeatFromCrawler(ctx, req.StuId, req.RoomId)
	if err != nil {
		return nil, err
	}

	var pbSeats []*pb.Seat
	for _, s := range seats {
		var ts []*pb.TimeSlot
		for _, t := range s.Ts {
			ts = append(ts, &pb.TimeSlot{
				Start: t.Start,
				End:   t.End,
				Owner: t.Owner,
			})
		}
		pbSeats = append(pbSeats, &pb.Seat{
			Name:     s.Name,
			DevId:    s.DevID,
			KindName: s.KindName,
			Ts:       ts,
		})
	}

	return &pb.GetSeatResponse{
		Seat: pbSeats,
	}, nil
}
