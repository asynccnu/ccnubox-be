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

func (b *commentUsecase) GetCommentsBySeatID(ctx context.Context, seatID int) ([]*Comment, error) {
	comments, err := b.repo.GetCommentsBySeatID(ctx, seatID)
	if err != nil {
		b.log.Errorf("Get comments failed (seat_id = %d)", seatID)
		return nil, err
	}

	return comments, nil
}

func (b *commentUsecase) DeleteComment(ctx context.Context, id int) (string, error) {
	message, err := b.repo.DeleteComment(ctx, id)
	if err != nil {
		b.log.Errorf("Deleted comments failed (id = %d)", id)
		return "", err
	}

	return message, nil
}
