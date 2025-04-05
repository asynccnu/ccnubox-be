package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"gorm.io/gorm"
)

type UserFeedTokenDAO interface {
	GetStudentIdAndTokensByCursor(ctx context.Context, lastID int64, limit int) (map[string][]string, int64, error)
	GetTokens(ctx context.Context, studentId string) ([]string, error)
	AddToken(ctx context.Context, studentId string, token string) error
	RemoveToken(ctx context.Context, studentId string, token string) error
}

type userFeedTokenDAO struct {
	gorm *gorm.DB
}

// NewUserFeedTokenDAO 创建一个新的 UserFeedTokenDAO 实例
func NewUserFeedTokenDAO(db *gorm.DB) UserFeedTokenDAO {
	return &userFeedTokenDAO{gorm: db}
}

func (dao *userFeedTokenDAO) GetStudentIdAndTokensByCursor(ctx context.Context, lastID int64, limit int) (map[string][]string, int64, error) {
	// 定义存储查询结果的结构体
	type UserTokens struct {
		ID        uint   `gorm:"column:id"`
		StudentId string `gorm:"column:student_id"`
		Token     string `gorm:"column:token"`
	}

	// 定义结果 map
	userTokenMap := make(map[string][]string)

	// 分页查询数据
	var userTokens []UserTokens
	query := dao.gorm.WithContext(ctx).
		Model(model.Token{}).
		Select("id, student_id, token").
		Order("id ASC"). // 按 id 排序，确保数据有序
		Limit(limit)

	// 如果提供了游标（lastID），则从该游标之后开始查询
	if lastID != -1 {
		query = query.Where("id > ?", lastID)
	}

	err := query.Scan(&userTokens).Error
	if err != nil {
		return nil, -1, err
	}

	// 如果没有数据，直接返回
	if len(userTokens) == 0 {
		return nil, -1, nil
	}

	// 将结果转换为 map 格式，并记录最后一个 ID
	var newLastID int64
	for _, ut := range userTokens {
		userTokenMap[ut.StudentId] = append(userTokenMap[ut.StudentId], ut.Token)
		newLastID = int64(ut.ID) // 更新最后一个 ID
	}

	return userTokenMap, newLastID, nil
}

func (dao *userFeedTokenDAO) GetTokens(ctx context.Context, studentId string) ([]string, error) {
	var tokens []string
	// 限制最多返回4个最新的 token
	err := dao.gorm.WithContext(ctx).
		Model(model.Token{}).
		Select("token").
		Where("student_id = ?", studentId).
		Order("created_at DESC"). // 按照 created_at 字段降序排列，确保获取最新的 token
		Limit(4).
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// 添加 Token
func (dao *userFeedTokenDAO) AddToken(ctx context.Context, studentId string, token string) error {
	newToken := model.Token{StudentId: studentId, Token: token}
	return dao.gorm.WithContext(ctx).Model(model.Token{}).Create(&newToken).Error
}

// 删除 Token
func (dao *userFeedTokenDAO) RemoveToken(ctx context.Context, studentId string, token string) error {
	return dao.gorm.WithContext(ctx).Model(model.Token{}).Where("student_id = ? and token = ?", studentId, token).Delete(&model.Token{}).Error
}
