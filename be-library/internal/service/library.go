package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	pb "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	"github.com/go-kratos/kratos/v2/log"
)

type LibraryService struct {
	pb.UnimplementedLibraryServer
	biz     biz.LibraryBiz
	log     *log.Helper
	conv    *Assembler
	comment biz.CommentRepo
}

func NewLibraryService(biz biz.LibraryBiz, logger log.Logger, comment biz.CommentRepo) *LibraryService {
	return &LibraryService{
		biz:     biz,
		log:     log.NewHelper(logger),
		conv:    NewAssembler(),
		comment: comment,
	}
}

func (ls *LibraryService) GetSeat(ctx context.Context, req *pb.GetSeatRequest) (*pb.GetSeatResponse, error) {
	data, err := ls.biz.GetSeat(ctx, req.StuId, req.RoomIds)
	if err != nil {
		return nil, err
	}
	return ls.conv.ConvertGetSeatResponse(data), nil
}

func (ls *LibraryService) ReserveSeat(ctx context.Context, req *pb.ReserveSeatRequest) (*pb.ReserveSeatResponse, error) {
	msg, err := ls.biz.ReserveSeat(ctx, req.StuId, req.DevId, req.Start, req.End)
	if err != nil {
		return nil, err
	}
	return &pb.ReserveSeatResponse{Message: msg}, nil
}

func (ls *LibraryService) GetSeatRecord(ctx context.Context, req *pb.GetSeatRecordRequest) (*pb.GetSeatRecordResponse, error) {
	records, err := ls.biz.GetSeatRecord(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	return &pb.GetSeatRecordResponse{
		Record: ls.conv.ConvertRecords(records),
	}, nil
}

func (ls *LibraryService) GetHistory(ctx context.Context, req *pb.GetHistoryRequest) (*pb.GetHistoryResponse, error) {
	history, err := ls.biz.GetHistory(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	return &pb.GetHistoryResponse{
		History: ls.conv.ConvertHistory(history),
	}, nil
}

func (ls *LibraryService) GetCreditPoint(ctx context.Context, req *pb.GetCreditPointRequest) (*pb.GetCreditPointResponse, error) {
	cp, err := ls.biz.GetCreditPoint(ctx, req.StuId)
	if err != nil {
		return nil, err
	}
	return &pb.GetCreditPointResponse{
		CreditSummary: &pb.CreditSummary{
			System: cp.Summary.System,
			Remain: cp.Summary.Remain,
			Total:  cp.Summary.Total,
		},
		CreditRecord: ls.conv.ConvertCreditRecords(cp.Records),
	}, nil
}

func (ls *LibraryService) GetDiscussion(ctx context.Context, req *pb.GetDiscussionRequest) (*pb.GetDiscussionResponse, error) {
	ds, err := ls.biz.GetDiscussion(ctx, req.StuId, req.ClassId, req.Date)
	if err != nil {
		return nil, err
	}
	return &pb.GetDiscussionResponse{
		Discussions: ls.conv.ConvertDiscussions(ds),
	}, nil
}

func (ls *LibraryService) SearchUser(ctx context.Context, req *pb.SearchUserRequest) (*pb.SearchUserResponse, error) {
	u, err := ls.biz.SearchUser(ctx, req.StuId, req.StudentId)
	if err != nil {
		return nil, err
	}
	return &pb.SearchUserResponse{
		Id:    u.ID,
		Pid:   u.Pid,
		Name:  u.Name,
		Label: u.Label,
	}, nil
}

func (ls *LibraryService) ReserveDiscussion(ctx context.Context, req *pb.ReserveDiscussionRequest) (*pb.ReserveDiscussionResponse, error) {
	msg, err := ls.biz.ReserveDiscussion(ctx, req.StuId, req.DevId, req.LabId, req.KindId, req.Title, req.Start, req.End, req.List)
	if err != nil {
		return nil, err
	}
	return &pb.ReserveDiscussionResponse{Message: msg}, nil
}

func (ls *LibraryService) CancelReserve(ctx context.Context, req *pb.CancelReserveRequest) (*pb.CancelReserveResponse, error) {
	msg, err := ls.biz.CancelReserve(ctx, req.StuId, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.CancelReserveResponse{Message: msg}, nil
}

func (ls *LibraryService) ReserveSeatRandomly(ctx context.Context, req *pb.ReserveSeatRandomlyRequest) (*pb.ReserveSeatRandomlyResponse, error) {
	//TODO: 简单修改完成 cicd 测试，记得修改
	RoomIDs := make([]string, 0)
	RoomIDs = append(RoomIDs, req.StuId)
	msg, err := ls.biz.ReserveSeatRandomly(ctx, req.StuId, req.Start, req.End, RoomIDs)
	if err != nil {
		return nil, err
	}
	return &pb.ReserveSeatRandomlyResponse{Message: msg}, nil
}

func (ls *LibraryService) CreateComment(ctx context.Context, req *pb.CreateCommentReq) (*pb.Resp, error) {
	msg, err := ls.comment.CreateComment(ctx, &biz.CreateCommentReq{
		Floor:    req.Floor,
		SeatID:   req.SeatId,
		Content:  req.Content,
		Rating:   int(req.Rating),
		Username: req.Username,
	})

	if err != nil {
		return nil, err
	}

	return &pb.Resp{
		Message: msg,
	}, nil
}

func (ls *LibraryService) GetComments(ctx context.Context, req *pb.GetCommentReq) (*pb.GetCommentResp, error) {
	comments, err := ls.comment.GetCommentsBySeatID(ctx, &biz.GetCommentReq{
		Floor:  req.Floor,
		SeatID: req.SeatId,
	})
	if err != nil {
		return nil, err
	}

	result := ls.conv.ConvertMessages(comments)

	return result, nil
}

func (ls *LibraryService) DeleteComment(ctx context.Context, req *pb.DeleteCommentReq) (*pb.Resp, error) {
	msg, err := ls.comment.DeleteComment(ctx, &biz.DeleteCommentReq{
		Username: req.Username,
		Floor:    req.Floor,
		SeatID:   req.SeatId,
	})
	if err != nil {
		return nil, err
	}

	return &pb.Resp{
		Message: msg,
	}, nil
}
