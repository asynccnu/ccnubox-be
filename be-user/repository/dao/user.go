package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-user/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
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
	err := dao.db.WithContext(ctx).Model(&model.User{}).Where("student_id = ?", u.StudentId).Save(u).Error
	if err != nil {
		return errorx.Errorf("dao: save user failed, student_id: %s, err: %w", u.StudentId, err)
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
			return nil, UserNotFound
		}
		return nil, errorx.Errorf("dao: find user failed, sid: %s, err: %w", sid, err)
	}

	return &u, nil
}
