package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type commentUsecase struct {
	repo CommentRepo
	log  *log.Helper
}

func NewCommentUsecase(repo CommentRepo, logger log.Logger) *commentUsecase {
	return &commentUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (b *commentUsecase) CreateComment(ctx context.Context, req CreateCommentReq) (string, error) {
	message, err := b.repo.CreateComment(ctx, &req)
	if err != nil {
		b.log.Errorf("created comment failed (seat_id = %s)", req.SeatID)
		return "", err
	}

	return message, nil
}

func (b *commentUsecase) GetCommentsBySeatID(ctx context.Context, req GetCommentReq) ([]*Comment, error) {
	comments, err := b.repo.GetCommentsBySeatID(ctx, &req)
	if err != nil {
		b.log.Errorf("Get comments failed (floor = %s,seat_id = %s)", req.Floor, req.SeatID)
		return nil, err
	}

	return comments, nil
}

func (b *commentUsecase) DeleteComment(ctx context.Context, req DeleteCommentReq) (string, error) {
	message, err := b.repo.DeleteComment(ctx, &req)
	if err != nil {
		b.log.Errorf("Deleted comments failed (username = %s,floor = %s,seatID = %s)", req.Username, req.Floor, req.SeatID)
		return "", err
	}

	return message, nil
}
