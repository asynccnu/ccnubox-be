package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"gorm.io/gorm"
)

// 用来对用户的feed数据进行处理

type UserFeedConfigDAO interface {
	FindOrCreateUserFeedConfig(ctx context.Context, studentId string) (*model.UserFeedConfig, error)
	SaveUserFeedConfig(ctx context.Context, req *model.UserFeedConfig) error
	SetConfigBit(config *uint16, position int)
	ClearConfigBit(config *uint16, position int)
	GetConfigBit(config uint16, position int) bool
	GetStudentIdsByCursor(ctx context.Context, lastID int64, limit int) ([]string, int64, error)
}

type userFeedConfigDAO struct {
	gorm *gorm.DB
}

// NewUserFeedConfigDAO 创建一个新的 UserFeedConfigDAO 实例
func NewUserFeedConfigDAO(db *gorm.DB) UserFeedConfigDAO {
	return &userFeedConfigDAO{gorm: db}
}

// FindOrCreateFeedAllowList 查找或创建 UserFeedConfig
func (dao *userFeedConfigDAO) FindOrCreateUserFeedConfig(ctx context.Context, studentId string) (*model.UserFeedConfig, error) {
	allowList := model.UserFeedConfig{StudentId: studentId}
	err := dao.gorm.WithContext(ctx).Model(model.UserFeedConfig{}).Where("student_id = ?", studentId).FirstOrCreate(&allowList).Error
	if err != nil {
		return nil, err
	}
	return &allowList, nil
}

// SaveFeedAllowList 保存 UserFeedConfig
func (dao *userFeedConfigDAO) SaveUserFeedConfig(ctx context.Context, req *model.UserFeedConfig) error {
	return dao.gorm.WithContext(ctx).Save(req).Error
}

// 设置指定位置的配置为 1
func (dao *userFeedConfigDAO) SetConfigBit(config *uint16, position int) {
	*config |= (1 << position)
}

// 设置指定位置的配置为 0
func (dao *userFeedConfigDAO) ClearConfigBit(config *uint16, position int) {
	*config &= ^(1 << position)
}

// 获取指定位置的配置值（返回 true 或 false）
func (dao *userFeedConfigDAO) GetConfigBit(config uint16, position int) bool {
	return (config & (1 << position)) != 0
}

func (dao *userFeedConfigDAO) GetStudentIdsByCursor(ctx context.Context, lastID int64, limit int) ([]string, int64, error) {
	// 创建查询条件：从 lastID 开始，限制数量为 limit
	var students []struct {
		ID        int64  `gorm:"column:id"`
		StudentId string `gorm:"column:student_id"`
	}

	// 查询数据库，假设有一个学生表，按 ID 排序
	query := dao.gorm.WithContext(ctx).Model(model.UserFeedConfig{}).Where("id > ?", lastID).Order("id ASC").Limit(limit)

	// 执行查询
	if err := query.Find(&students).Error; err != nil {
		// 查询失败，返回错误
		return nil, 0, err
	}

	// 如果没有找到数据，返回空切片
	if len(students) == 0 {
		return nil, 0, nil
	}

	// 提取学生 ID 列表
	var studentIds []string
	for _, student := range students {
		studentIds = append(studentIds, student.StudentId)
	}

	// 设置新的 lastID，表示下一次查询的起始 ID
	newLastID := students[len(students)-1].ID

	return studentIds, newLastID, nil
}
