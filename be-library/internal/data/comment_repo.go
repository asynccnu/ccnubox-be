package data

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/dao"
	"github.com/go-kratos/kratos/v2/log"
)

type CommentRepo struct {
	dao  *dao.CommentDAO
	log  *log.Helper
	conv *Assembler
}

func NewCommentRepo(commentDAO *dao.CommentDAO, logger log.Logger, conv *Assembler) biz.CommentRepo {
	return &CommentRepo{
		log:  log.NewHelper(logger),
		dao:  commentDAO,
		conv: conv,
	}
}

func (r CommentRepo) CreateComment(ctx context.Context, req *biz.CreateCommentReq) (string, error) {
	comment := &DO.Comment{
		SeatID:    req.SeatID,
		Content:   req.Content,
		Rating:    req.Rating,
		Username:  req.Username,
		CreatedAt: time.Now(),
	}

	err := r.dao.CreateComment(ctx, comment)
	if err != nil {
		return "", err
	}

	return "success", nil
}

func (r CommentRepo) GetCommentsBySeatID(ctx context.Context, seatID int) ([]*biz.Comment, error) {
	comments, err := r.dao.GetCommentsBySeatID(ctx, seatID)
	if err != nil {
		return nil, err
	}
	return r.conv.ConvertDOCommentBiz(comments), nil
}

func (r CommentRepo) DeleteComment(ctx context.Context, id int) (string, error) {
	err := r.dao.DeleteComment(ctx, id)
	if err != nil {
		return "", err
	}

	return "success", nil
}
