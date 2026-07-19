package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	SeatID    string    `gorm:"index;not null"`
	Username  string    `gorm:"index;not null"`
	Content   string    `gorm:"type:text;not null"`
	Rating    int64     `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type CommentDAO interface {
	Create(ctx context.Context, comment *Comment) error
	FindBySeatID(ctx context.Context, seatID string) ([]*Comment, error)
	DeleteByID(ctx context.Context, id int64) error
}

type commentDAO struct {
	db *gorm.DB
}

func NewCommentDAO(db *gorm.DB) CommentDAO {
	return &commentDAO{db: db}
}

func (d *commentDAO) Create(ctx context.Context, comment *Comment) error {
	return d.db.WithContext(ctx).Create(comment).Error
}

func (d *commentDAO) FindBySeatID(ctx context.Context, seatID string) ([]*Comment, error) {
	var comments []*Comment
	err := d.db.WithContext(ctx).
		Where("seat_id = ?", seatID).
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (d *commentDAO) DeleteByID(ctx context.Context, id int64) error {
	result := d.db.WithContext(ctx).Delete(&Comment{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
