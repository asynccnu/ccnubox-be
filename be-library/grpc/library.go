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
	comment    service.CommentService
}

func NewLibraryGrpcService(ss service.SeatService, ds service.DiscussionService, cs service.CommentService) *LibraryServiceServer {
	return &LibraryServiceServer{
		seat:       ss,
		discussion: ds,
		comment:    cs,
	}
}

func (l *LibraryServiceServer) Register(server grpc.ServiceRegistrar) {
	v1.RegisterLibraryServiceServer(server, l)
}

func (l *LibraryServiceServer) GetSeat(ctx context.Context, req *v1.GetSeatRequest) (*v1.GetSeatResponse, error) {
	return l.seat.GetSeat(ctx, req)
}

func (l *LibraryServiceServer) ReserveSeat(ctx context.Context, req *v1.ReserveSeatRequest) (*v1.ReserveSeatResponse, error) {
	return l.seat.ReserveSeat(ctx, req)
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

func (l *LibraryServiceServer) CancelReserve(ctx context.Context, req *v1.CancelReserveRequest) (*v1.CancelReserveResponse, error) {
	return l.seat.CancelReserve(ctx, req)
}

func (l *LibraryServiceServer) ReserveSeatRandomly(ctx context.Context, req *v1.ReserveSeatRandomlyRequest) (*v1.ReserveSeatRandomlyResponse, error) {
	return l.seat.ReserveSeatRandomly(ctx, req)
}

func (l *LibraryServiceServer) CreateComment(ctx context.Context, req *v1.CreateCommentReq) (*v1.Resp, error) {
	return l.comment.CreateComment(ctx, req)
}

func (l *LibraryServiceServer) GetComments(ctx context.Context, req *v1.ID) (*v1.GetCommentResp, error) {
	return l.comment.GetComments(ctx, req)
}

func (l *LibraryServiceServer) DeleteComment(ctx context.Context, req *v1.ID) (*v1.Resp, error) {
	return l.comment.DeleteComment(ctx, req)
}
