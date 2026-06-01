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

func (s *ClassListService) AddClass(ctx context.Context, stuID, name, durClass, where, teacher string, weeks int64, semester, year string, day int64, credit *float64) (id, msg string, err error) {
	logh := s.log.WithContext(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
	)

	if !tool.CheckSY(semester, year) || weeks <= 0 || !tool.CheckIfThisYear(year, semester) {
		logh.Warn("add class param invalid",
			logger.Int64("weeks", weeks),
			logger.Int64("day", day),
		)
		return "", "", errcode.ErrParam
	}

	weekDur := tool.FormatWeeks(tool.ParseWeeks(weeks))
	classInfo := &model.ClassInfoBO{
		Day:          day,
		Teacher:      teacher,
		Where:        where,
		ClassWhen:    durClass,
		WeekDuration: weekDur,
		Classname:    name,
		Weeks:        weeks,
		Semester:     semester,
		Year:         year,
		JxbId:        "unavailable",
	}
	if credit != nil {
		classInfo.Credit = *credit
	}
	classInfo.UpdateID()

	if err := s.clu.AddClass(ctx, stuID, classInfo); err != nil {
		return "", "", err
	}
	return classInfo.ID, "成功添加", nil
}

func (s *ClassListService) DeleteClass(ctx context.Context, stuID, year, semester, classID string) (string, error) {
	logh := s.log.WithContext(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
		logger.String("class_id", classID),
	)

	if !tool.CheckSY(semester, year) || classID == "" {
		logh.Warn("delete class param invalid")
		return "", errcode.ErrParam
	}

	if err := s.clu.DeleteClass(ctx, stuID, year, semester, classID); err != nil {
		return "删除课程失败", err
	}
	return "删除课程成功", nil
}

func (s *ClassListService) UpdateClass(ctx context.Context, stuID, year, semester, classID string, name, durClass, where, teacher *string, weeks, day *int64, credit *float64) (string, string, error) {
	logh := s.log.WithContext(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
		logger.String("class_id", classID),
	)

	if !tool.CheckSY(semester, year) || classID == "" {
		logh.Warn("update class param invalid")
		return "", "", errcode.ErrParam
	}
	if weeks != nil && *weeks <= 0 {
		logh.Warn("update class weeks invalid", logger.Int64("weeks", *weeks))
		return "", "", errcode.ErrParam
	}
	if day != nil && (*day < 1 || *day > 7) {
		logh.Warn("update class day invalid", logger.Int64("day", *day))
		return "", "", errcode.ErrParam
	}

	newClassID, err := s.clu.UpdateClass(ctx, stuID, year, semester, classID, name, durClass, where, teacher, weeks, day, credit)
	if err != nil {
		return "", "修改失败", err
	}
	return newClassID, "成功修改", nil
}

func (s *ClassListService) UpdateClassNote(ctx context.Context, stuID, year, semester, classID, note string) (string, error) {
	logh := s.log.WithContext(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
		logger.String("class_id", classID),
	)

	if !tool.CheckSY(semester, year) || classID == "" {
		logh.Warn("update class note param invalid")
		return "", errcode.ErrParam
	}

	if err := s.clu.UpdateClassNote(ctx, stuID, year, semester, classID, note); err != nil {
		return "更新课程备注失败", err
	}
	return "更新课程备注成功", nil
}

func (s *ClassListService) DeleteClassNote(ctx context.Context, stuID, year, semester, classID string) (string, error) {
	logh := s.log.WithContext(ctx).With(
		logger.String("stu_id", stuID),
		logger.String("year", year),
		logger.String("semester", semester),
		logger.String("class_id", classID),
	)

	if !tool.CheckSY(semester, year) || classID == "" {
		logh.Warn("delete class note param invalid")
		return "", errcode.ErrParam
	}

	if err := s.clu.UpdateClassNote(ctx, stuID, year, semester, classID, ""); err != nil {
		return "删除课程备注失败", err
	}
	return "删除课程备注成功", nil
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
