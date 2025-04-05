package service

import (
	"context"
	departmentv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/department/v1"
	"github.com/asynccnu/ccnubox-be/be-department/domain"
	"github.com/asynccnu/ccnubox-be/be-department/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-department/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-department/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-department/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-department/repository/model"
	"github.com/jinzhu/copier"
	"time"
)

type DepartmentService interface {
	GetDepartments(ctx context.Context) ([]*domain.Department, error)
	SaveDepartment(ctx context.Context, req *domain.Department) error
	DelDepartment(ctx context.Context, id uint) error
}

type departmentService struct {
	dao   dao.DepartmentDAO
	cache cache.DepartmentCache
	l     logger.Logger
}

func NewDepartmentService(dao dao.DepartmentDAO, cache cache.DepartmentCache, l logger.Logger) DepartmentService {
	return &departmentService{dao: dao, cache: cache, l: l}
}

// 定义错误函数
var (
	GET_DEPARTMENT_ERROR = func(err error) error {
		return errorx.New(departmentv1.ErrorGetDepartmentError("获取部门失败"), "dao", err)
	}

	DEL_DEPARTMENT_ERROR = func(err error) error {
		return errorx.New(departmentv1.ErrorDelDepartmentError("删除部门失败"), "dao", err)
	}

	SAVE_DEPARTMENT_ERROR = func(err error) error {
		return errorx.New(departmentv1.ErrorSaveDepartmentError("保存部门失败"), "dao", err)
	}
)

func (s *departmentService) GetDepartments(ctx context.Context) ([]*domain.Department, error) {
	// 尝试从缓存获取
	res, err := s.cache.GetDepartments(ctx)
	if err == nil {
		return res, nil
	}
	s.l.Info("从缓存获取部门失败", logger.FormatLog("cache", err)...)

	// 如果缓存中不存在则从数据库获取
	dps, err := s.dao.GetDepartments(ctx)
	if err != nil {
		return nil, GET_DEPARTMENT_ERROR(err)
	}

	// 类型转换
	err = copier.Copy(&res, &dps)
	if err != nil {
		s.l.Error("类型转换失败", logger.FormatLog("error", err)...)
		return nil, GET_DEPARTMENT_ERROR(err)
	}

	return res, nil
}

func (s *departmentService) SaveDepartment(ctx context.Context, req *domain.Department) error {
	// 查找部门是否存在
	department, err := s.dao.FindDepartment(ctx, req.ID)
	if department != nil {
		department.Name = req.Name
		department.Phone = req.Phone
		department.Place = req.Place
		department.Time = req.Time
	} else {
		department = s.toEntity(req)
	}

	// 存储或者更新部门
	err = s.dao.SaveDepartment(ctx, department)
	if err != nil {
		s.l.Error("保存部门失败", logger.FormatLog("dao", err)...)
		return SAVE_DEPARTMENT_ERROR(err)
	}

	// 异步写入缓存，牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// 获取所有部门并写入缓存
		dps := []*domain.Department{}
		res, er := s.dao.GetDepartments(ct)
		if er != nil {
			s.l.Error("获取部门失败", logger.FormatLog("dao", er)...)
			return
		}

		err := copier.Copy(&dps, &res)
		if err != nil {
			s.l.Error("类型转换失败", logger.FormatLog("error", err)...)
			return
		}

		// 写缓存
		er = s.cache.SetDepartments(ct, dps)
		if er != nil {
			s.l.Error("回写部门资源失败", logger.FormatLog("cache", er)...)
		}
	}()

	return nil
}

func (s *departmentService) DelDepartment(ctx context.Context, id uint) error {
	err := s.dao.DelDepartment(ctx, id)
	if err != nil {
		return DEL_DEPARTMENT_ERROR(err)
	}

	// 异步写入缓存，牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// 获取部门并更新缓存
		dps := []*domain.Department{}
		res, er := s.dao.GetDepartments(ct)
		if er != nil {
			s.l.Error("获取部门失败", logger.FormatLog("dao", er)...)
			return
		}

		err := copier.Copy(&dps, &res)
		if err != nil {
			s.l.Error("类型转换失败", logger.FormatLog("error", err)...)
			return
		}

		// 设置缓存
		er = s.cache.SetDepartments(ct, dps)
		if er != nil {
			s.l.Error("回写部门资源失败", logger.FormatLog("cache", er)...)
		}
	}()

	return nil
}

func (s *departmentService) toDomain(u *model.Department) *domain.Department {
	return &domain.Department{
		Name:  u.Name,
		Phone: u.Phone,
		Place: u.Place,
		Time:  u.Time,
		Model: u.Model,
	}
}

func (s *departmentService) toEntity(u *domain.Department) *model.Department {
	return &model.Department{
		Name:  u.Name,
		Phone: u.Phone,
		Place: u.Place,
		Time:  u.Time,
		Model: u.Model,
	}
}
