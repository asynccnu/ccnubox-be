package dao

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
	"github.com/asynccnu/ccnubox-be/be-library/internal/errcode"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type CommentDAO struct {
	db *gorm.DB
}

func NewCommentDAO(db *gorm.DB) *CommentDAO {
	return &CommentDAO{db: db}
}

func (d *CommentDAO) CreateComment(ctx context.Context, comment *DO.Comment) error {
	if comment.Rating < 0 || comment.Rating > 5 {
		log.Errorf("Mysql:create %v in %s failed: %v", comment.Rating, "comment", fmt.Errorf("rating must between 0 and 5"))
		return errcode.ErrCreateComment
	}

	if err := d.db.WithContext(ctx).Create(comment).Error; err != nil {
		log.Errorf("CreateComment failed: %v | stu_id=%v seat_id=%v", err, comment.Username, comment.SeatID)
		return errcode.ErrCreateComment
	}

	return nil
}

func (d *CommentDAO) GetCommentsBySeatID(ctx context.Context, floor, seatID string) ([]*DO.Comment, error) {
	var comments []*DO.Comment
	err := d.db.WithContext(ctx).
		Where("floor = ? AND seat_id = ?", floor, seatID).
		Order("created_at DESC").
		Find(&comments).Error
	if err != nil {
		log.Errorf("GetCommentsBySeatID failed: %v | seat_id=%v", err, seatID)
		return nil, err
	}

	return comments, nil
}

func (d *CommentDAO) DeleteComment(ctx context.Context, username, floor, seatID string) error {
	return d.db.WithContext(ctx).
		Where("username = ? AND floor = ? AND seat_id = ?", username, floor, seatID).
		Delete(&DO.Comment{}).Error
}
