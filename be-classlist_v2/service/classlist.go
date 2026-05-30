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

func (s *ClassListService) GetSchoolDay(ctx context.Context) (holidayTime, schoolTime string, err error) {
	logh := s.log.WithContext(ctx)

	if s.conf == nil || s.conf.ClassListConf == nil {
		logh.Error("classlist school day config is empty")
		return "", "", errcode.ErrConfig
	}

	holidayTime = s.conf.ClassListConf.HolidayTime
	schoolTime = s.conf.ClassListConf.SchoolTime
	if holidayTime == "" || schoolTime == "" {
		logh.Error("classlist school day config is incomplete",
			logger.String("holidayTime", holidayTime),
			logger.String("schoolTime", schoolTime),
		)
		return "", "", errcode.ErrConfig
	}

	holiday, err := time.ParseInLocation("2006-01-02", holidayTime, time.Local)
	if err != nil {
		logh.Error("invalid classlist holidayTime config",
			logger.String("holidayTime", holidayTime),
			logger.Error(err),
		)
		return "", "", errcode.ErrConfig
	}
	school, err := time.ParseInLocation("2006-01-02", schoolTime, time.Local)
	if err != nil {
		logh.Error("invalid classlist schoolTime config",
			logger.String("schoolTime", schoolTime),
			logger.Error(err),
		)
		return "", "", errcode.ErrConfig
	}
	if !school.Before(holiday) {
		logh.Error("classlist schoolTime must be before holidayTime",
			logger.String("schoolTime", schoolTime),
			logger.String("holidayTime", holidayTime),
		)
		return "", "", errcode.ErrConfig
	}

	return holidayTime, schoolTime, nil
}
