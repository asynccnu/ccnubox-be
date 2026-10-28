package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

type FeedTokenDAO interface {
	GetTokens(ctx context.Context, studentId string) ([]string, error)
	GetTokensBatch(ctx context.Context, studentIDs []string) (map[string][]string, error)
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

// GetTokensBatch 兼容 MySQL 5.7；调用方按最多 10 人分组，逐人查询最近 4 条有效 token。
func (dao *feedTokenDAO) GetTokensBatch(ctx context.Context, studentIDs []string) (map[string][]string, error) {
	ids := uniqueStudentIDs(studentIDs)
	result := make(map[string][]string, len(ids))
	for _, studentID := range ids {
		tokens, err := dao.GetTokens(ctx, studentID)
		if err != nil {
			// 不允许使用部分结果继续认领或抑制任务。
			return nil, errorx.Errorf("dao: get tokens batch failed, count: %d, err: %w", len(ids), err)
		}
		result[studentID] = tokens
	}
	return result, nil
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
