package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-user/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// UserNotFound 定义业务层可感知的错误
	UserNotFound = errorx.New("dao: user not found")
)

type UserDAO interface {
	FindByStudentId(ctx context.Context, sid string) (*model.User, error)
	Save(ctx context.Context, u *model.User) error
}

type GORMUserDAO struct {
	db *gorm.DB
}

// NewGORMUserDAO 构建数据库操作实例
func NewGORMUserDAO(db *gorm.DB) UserDAO {
	return &GORMUserDAO{db: db}
}

// Save 实现更新或创建逻辑
func (dao *GORMUserDAO) Save(ctx context.Context, u *model.User) error {
	// 使用 OnConflict 处理并发写入冲突，确保 student_id 唯一性下的正确保存
	err := dao.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "student_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"password", "updated_at"}),
	}).Create(u).Error

	if err != nil {
		return errorx.Errorf("dao: save user failed, sid: %s, err: %w", u.StudentId, err)
	}
	return nil
}

// FindByStudentId 根据学号查询用户信息
func (dao *GORMUserDAO) FindByStudentId(ctx context.Context, sid string) (*model.User, error) {
	var u model.User
	err := dao.db.WithContext(ctx).
		Where("student_id = ?", sid).
		First(&u).Error

	if err != nil {
		if errorx.Is(err, gorm.ErrRecordNotFound) {
			// 包装业务错误，并保留底层 gorm 错误链
			return nil, errorx.Errorf("%w: student_id %s", UserNotFound, sid)
		}
		return nil, errorx.Errorf("dao: find user failed, sid: %s, err: %w", sid, err)
	}

	return &u, nil
}
