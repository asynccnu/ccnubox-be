package biz

import (
	"context"
	"time"
)

type Comment struct {
	ID        int       // 评论ID
	Floor     string    // 楼层
	SeatID    string    // 关联座位
	Username  string    // 发表评论的用户
	Content   string    // 评论内容
	Rating    int       // 评分（1-5）
	CreatedAt time.Time // 创建时间
}

type CommentRepo interface {
	CreateComment(ctx context.Context, req *CreateCommentReq) (string, error)
	GetCommentsBySeatID(ctx context.Context, req *GetCommentReq) ([]*Comment, error)
	DeleteComment(ctx context.Context, req *DeleteCommentReq) (string, error)
}

type CreateCommentReq struct {
	Floor    string
	SeatID   string
	Content  string
	Rating   int
	Username string
}

type GetCommentReq struct {
	Floor  string
	SeatID string
}

type DeleteCommentReq struct {
	Username string
	Floor    string
	SeatID   string
}
