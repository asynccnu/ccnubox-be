package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

type FeedTokenDAO interface {
	GetTokens(ctx context.Context, studentId string) ([]string, error)
	AddToken(ctx context.Context, studentId string, token string) error
	RemoveToken(ctx context.Context, studentId string, token string) error
}

type feedTokenDAO struct {
	gorm *gorm.DB
}

// NewUserFeedTokenDAO 创建一个新的 FeedTokenDAO 实例
func NewUserFeedTokenDAO(db *gorm.DB) FeedTokenDAO {
	return &feedTokenDAO{gorm: db}
}

func (dao *feedTokenDAO) GetTokens(ctx context.Context, studentId string) ([]string, error) {
	var tokens []string
	err := dao.gorm.WithContext(ctx).
		Model(model.FeedUserToken{}).
		Select("token").
		Where("student_id = ?", studentId).
		Order("created_at DESC, id DESC").
		Limit(4).
		Find(&tokens).Error
	if err != nil {
		return nil, errorx.Errorf("dao: get tokens failed, sid: %s, err: %w", studentId, err)
	}
	return tokens, nil
}

// AddToken 添加 FeedUserToken
func (dao *feedTokenDAO) AddToken(ctx context.Context, studentId string, token string) error {
	newToken := model.FeedUserToken{StudentId: studentId, Token: token}
	err := dao.gorm.WithContext(ctx).Model(model.FeedUserToken{}).Create(&newToken).Error
	if err != nil {
		return errorx.Errorf("dao: add feed token failed, sid: %s, err: %w", studentId, err)
	}
	return nil
}

// RemoveToken 删除 FeedUserToken
func (dao *feedTokenDAO) RemoveToken(ctx context.Context, studentId string, token string) error {
	err := dao.gorm.WithContext(ctx).
		Model(model.FeedUserToken{}).
		Where("student_id = ? and token = ?", studentId, token).
		Delete(&model.FeedUserToken{}).Error
	if err != nil {
		return errorx.Errorf("dao: remove feed token failed, sid: %s, err: %w", studentId, err)
	}
	return nil
}
