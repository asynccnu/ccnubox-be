package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/service"

	v1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	"google.golang.org/grpc"
)

type LibraryServiceServer struct {
	v1.UnimplementedLibraryServiceServer
	seat       service.SeatService
	discussion service.DiscussionService
}

func NewLibraryGrpcService(ss service.SeatService, ds service.DiscussionService) *LibraryServiceServer {
	return &LibraryServiceServer{
		seat:       ss,
		discussion: ds,
	}
}

func (l *LibraryServiceServer) Register(server grpc.ServiceRegistrar) {
	v1.RegisterLibraryServiceServer(server, l)
}

func (l *LibraryServiceServer) GetSeat(ctx context.Context, req *v1.GetSeatRequest) (*v1.GetSeatResponse, error) {
	return l.seat.GetSeat(ctx, req)
}

func (l *LibraryServiceServer) GetSeatRecord(ctx context.Context, req *v1.GetSeatRecordRequest) (*v1.GetSeatRecordResponse, error) {
	return l.seat.GetSeatRecord(ctx, req)
}

func (l *LibraryServiceServer) GetDiscussion(ctx context.Context, req *v1.GetDiscussionRequest) (*v1.GetDiscussionResponse, error) {
	return l.discussion.GetDiscussion(ctx, req)
}

func (l *LibraryServiceServer) ReserveDiscussion(ctx context.Context, req *v1.ReserveDiscussionRequest) (*v1.ReserveDiscussionResponse, error) {
	return l.discussion.ReserveDiscussion(ctx, req)
}
