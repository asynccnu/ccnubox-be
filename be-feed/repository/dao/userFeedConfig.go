package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

// FeedUserConfigDAO 用来对用户的feed数据进行处理
type FeedUserConfigDAO interface {
	FindOrCreateUserFeedConfig(ctx context.Context, studentId string) (*model.FeedUserConfig, error)
	SaveUserFeedConfig(ctx context.Context, req *model.FeedUserConfig) error
	SetConfigBit(config *uint16, position int)
	ClearConfigBit(config *uint16, position int)
	GetConfigBit(config uint16, position int) bool
	GetStudentIdsByCursor(ctx context.Context, lastID int64, limit int) ([]string, int64, error)
}

type feedUserConfigDAO struct {
	gorm *gorm.DB
}

// NewFeedUserConfigDAO 创建一个新的 FeedUserConfigDAO 实例
func NewFeedUserConfigDAO(db *gorm.DB) FeedUserConfigDAO {
	return &feedUserConfigDAO{gorm: db}
}

// FindOrCreateUserFeedConfig 查找或创建 FeedUserConfig
func (dao *feedUserConfigDAO) FindOrCreateUserFeedConfig(ctx context.Context, studentId string) (*model.FeedUserConfig, error) {
	allowList := model.FeedUserConfig{StudentId: studentId}
	err := dao.gorm.WithContext(ctx).Model(model.FeedUserConfig{}).
		Where("student_id = ?", studentId).
		FirstOrCreate(&allowList).Error
	if err != nil {
		return nil, errorx.Errorf("dao: find or create user feed config failed, sid: %s, err: %w", studentId, err)
	}
	return &allowList, nil
}

// SaveUserFeedConfig 保存 FeedUserConfig
func (dao *feedUserConfigDAO) SaveUserFeedConfig(ctx context.Context, req *model.FeedUserConfig) error {
	err := dao.gorm.WithContext(ctx).Save(req).Error
	if err != nil {
		return errorx.Errorf("dao: save user feed config failed, sid: %s, err: %w", req.StudentId, err)
	}
	return nil
}

// 设置指定位置的配置为 1
func (dao *feedUserConfigDAO) SetConfigBit(config *uint16, position int) {
	*config |= (1 << position)
}

// 设置指定位置的配置为 0
func (dao *feedUserConfigDAO) ClearConfigBit(config *uint16, position int) {
	*config &= ^(1 << position)
}

// 获取指定位置的配置值（返回 true 或 false）
func (dao *feedUserConfigDAO) GetConfigBit(config uint16, position int) bool {
	return (config & (1 << position)) != 0
}

func (dao *feedUserConfigDAO) GetStudentIdsByCursor(ctx context.Context, lastID int64, limit int) ([]string, int64, error) {
	var students []struct {
		ID        int64  `gorm:"column:id"`
		StudentId string `gorm:"column:student_id"`
	}

	query := dao.gorm.WithContext(ctx).Model(model.FeedUserConfig{}).
		Where("id > ?", lastID).
		Order("id ASC").
		Limit(limit)

	if err := query.Find(&students).Error; err != nil {
		return nil, 0, errorx.Errorf("dao: get student ids by cursor failed, lastID: %d, limit: %d, err: %w", lastID, limit, err)
	}

	if len(students) == 0 {
		return nil, 0, nil
	}

	var studentIds []string
	for _, student := range students {
		studentIds = append(studentIds, student.StudentId)
	}

	newLastID := students[len(students)-1].ID

	return studentIds, newLastID, nil
}
