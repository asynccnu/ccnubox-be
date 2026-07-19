package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-library/repository/dao"
	v1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/library/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

type CommentService interface {
	CreateComment(ctx context.Context, req *v1.CreateCommentReq) (*v1.Resp, error)
	GetComments(ctx context.Context, req *v1.ID) (*v1.GetCommentResp, error)
	DeleteComment(ctx context.Context, req *v1.ID) (*v1.Resp, error)
}

type commentService struct {
	dao dao.CommentDAO
}

func NewCommentService(commentDAO dao.CommentDAO) CommentService {
	return &commentService{dao: commentDAO}
}

func (s *commentService) CreateComment(ctx context.Context, req *v1.CreateCommentReq) (*v1.Resp, error) {
	if req == nil || strings.TrimSpace(req.SeatId) == "" || strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Content) == "" {
		return nil, errorx.New("seat_id, username and content are required")
	}
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errorx.New("rating must be between 1 and 5")
	}
	comment := &dao.Comment{
		SeatID:   strings.TrimSpace(req.SeatId),
		Username: strings.TrimSpace(req.Username),
		Content:  strings.TrimSpace(req.Content),
		Rating:   req.Rating,
	}
	if err := s.dao.Create(ctx, comment); err != nil {
		return nil, errorx.Errorf("create comment: %w", err)
	}
	return &v1.Resp{Message: "success"}, nil
}

func (s *commentService) GetComments(ctx context.Context, req *v1.ID) (*v1.GetCommentResp, error) {
	if req == nil || req.Id <= 0 {
		return nil, errorx.New("seat id must be positive")
	}
	comments, err := s.dao.FindBySeatID(ctx, strconv.FormatInt(req.Id, 10))
	if err != nil {
		return nil, errorx.Errorf("get comments: %w", err)
	}
	result := make([]*v1.Comment, 0, len(comments))
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		result = append(result, &v1.Comment{
			Id:        comment.ID,
			SeatId:    comment.SeatID,
			Username:  comment.Username,
			Content:   comment.Content,
			Rating:    comment.Rating,
			CreatedAt: comment.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &v1.GetCommentResp{Comment: result}, nil
}

func (s *commentService) DeleteComment(ctx context.Context, req *v1.ID) (*v1.Resp, error) {
	if req == nil || req.Id <= 0 {
		return nil, errorx.New("comment id must be positive")
	}
	if err := s.dao.DeleteByID(ctx, req.Id); err != nil {
		return nil, errorx.Errorf("delete comment: %w", err)
	}
	return &v1.Resp{Message: "success"}, nil
}
