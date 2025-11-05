package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
	"gorm.io/gorm"
)

type CommentDAO struct {
	db *gorm.DB
}

func NewCommentDAO(db *gorm.DB) *CommentDAO {
	return &CommentDAO{db: db}
}

func (d *CommentDAO) CreateComment(ctx context.Context, comment *DO.Comment) error {
	return d.db.WithContext(ctx).Create(comment).Error
}

func (d *CommentDAO) GetCommentsBySeatID(ctx context.Context, seatID int) ([]*DO.Comment, error) {
	var comments []*DO.Comment
	err := d.db.WithContext(ctx).
		Where("seat_id = ?", seatID).
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (d *CommentDAO) DeleteComment(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&DO.Comment{}, id).Error
}
