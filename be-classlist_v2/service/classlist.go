package service

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/model"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/usecase"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/pkg/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	ctool "github.com/asynccnu/ccnubox-be/common/tool"
)

type ClassListService struct {
	clu  *usecase.ClassUsecase
	conf *conf.ServerConf
	log  logger.Logger
}

func NewClasserService(clu *usecase.ClassUsecase, conf *conf.ServerConf, l logger.Logger) *ClassListService {
	return &ClassListService{
		clu:  clu,
		conf: conf,
		log:  l,
	}
}

// GetClass 获取课表
// 业务默认：year/semester 为空时使用当前学年学期
// 业务校验：CheckSY 校验学年学期格式
func (s *ClassListService) GetClass(ctx context.Context, stuID, year, semester string, refresh bool) ([]*model.ClassInfoBO, *time.Time, error) {
	hlog := s.log.WithContext(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
	)

	// 默认值填充
	defaultYear, defaultSemester := ctool.GetCurrentAcademicYearAndSemesterStr(time.Now())
	if year == "" {
		year = defaultYear
		hlog.Warnf("year 参数为空，使用默认值 %s", year)
	}
	if semester == "" {
		semester = defaultSemester
		hlog.Warnf("semester 参数为空，使用默认值 %s", semester)
	}

	// 参数校验
	if !tool.CheckSY(semester, year) {
		return nil, nil, errcode.ErrParam
	}

	return s.clu.GetClasses(ctx, stuID, year, semester, refresh)
}
