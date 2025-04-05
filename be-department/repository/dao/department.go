package dao

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-department/repository/model"
	"gorm.io/gorm"
)

type DepartmentDAO interface {
	GetDepartments(ctx context.Context) ([]*model.Department, error)
	FindDepartment(ctx context.Context, Id uint) (*model.Department, error)
	SaveDepartment(ctx context.Context, req *model.Department) error
	DelDepartment(ctx context.Context, Id uint) error
}

type departmentDAO struct {
	gorm *gorm.DB
}

func NewMysqlDepartmentDAO(db *gorm.DB) DepartmentDAO {
	return &departmentDAO{gorm: db}
}

func (dao departmentDAO) GetDepartments(ctx context.Context) ([]*model.Department, error) {
	var s []*model.Department
	err := dao.gorm.WithContext(ctx).Find(&s).Error
	return s, err
}

func (dao departmentDAO) SaveDepartment(ctx context.Context, req *model.Department) error {
	return dao.gorm.WithContext(ctx).Save(req).Error
}

func (dao departmentDAO) FindDepartment(ctx context.Context, Id uint) (*model.Department, error) {
	d := model.Department{}
	err := dao.gorm.WithContext(ctx).Where("id=?", Id).First(&d).Error
	return &d, err
}

func (dao departmentDAO) AddDepartment(ctx context.Context, req *model.Department) error {
	return dao.gorm.WithContext(ctx).Create(req).Error
}

func (dao departmentDAO) DelDepartment(ctx context.Context, Id uint) error {
	return dao.gorm.WithContext(ctx).Where("id=?", Id).Delete(&model.Department{}).Error
}
