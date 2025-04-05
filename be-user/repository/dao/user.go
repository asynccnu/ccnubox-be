package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-user/repository/model"
	"gorm.io/gorm"
)

type UserDAO interface {
	FindByStudentId(ctx context.Context, sid string) (*model.User, error)
	Save(ctx context.Context, u *model.User) error
}

type GORMUserDAO struct {
	db *gorm.DB
}

func (dao *GORMUserDAO) Save(ctx context.Context, u *model.User) error {
	return dao.db.WithContext(ctx).Model(&model.User{}).Where("student_id = ?", u.StudentId).Save(u).Error
}

func (dao *GORMUserDAO) FindByStudentId(ctx context.Context, sid string) (*model.User, error) {
	var u model.User
	err := dao.db.WithContext(ctx).Model(&model.User{}).Where("student_id = ?", sid).First(&u).Error
	return &u, err
}

func NewGORMUserDAO(db *gorm.DB) UserDAO {
	return &GORMUserDAO{db: db}
}
