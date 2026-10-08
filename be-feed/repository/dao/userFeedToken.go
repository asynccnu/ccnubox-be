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

// GetTokensBatch 依赖窗口函数（MySQL 8.0+），每个用户最多返回最近 4 条有效 token。
func (dao *feedTokenDAO) GetTokensBatch(ctx context.Context, studentIDs []string) (map[string][]string, error) {
	result := make(map[string][]string)
	studentIDs = uniqueStudentIDs(studentIDs)
	if len(studentIDs) == 0 {
		return result, nil
	}
	var tokens []struct {
		StudentId string
		Token     string
	}
	// 模型的默认作用域在排名前过滤软删除记录，避免其占用前 4 个名额。
	ranked := dao.gorm.WithContext(ctx).Model(&model.FeedUserToken{}).
		Select("student_id, token, ROW_NUMBER() OVER (PARTITION BY student_id ORDER BY created_at DESC, id DESC) AS token_rank").
		Where("student_id IN ?", studentIDs)
	err := dao.gorm.WithContext(ctx).Table("(?) AS ranked_tokens", ranked).
		Select("student_id, token").Where("token_rank <= ?", 4).
		Order("student_id ASC, token_rank ASC").Scan(&tokens).Error
	if err != nil {
		return nil, errorx.Errorf("dao: get tokens batch failed, count: %d, err: %w", len(studentIDs), err)
	}
	for _, token := range tokens {
		result[token.StudentId] = append(result[token.StudentId], token.Token)
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
