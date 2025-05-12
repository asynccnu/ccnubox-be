package service

import (
	"context"
	calendarv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/calendar/v1"
	"github.com/asynccnu/ccnubox-be/be-calendar/domain" // 替换为calendar的domain路径
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/logger"       // 替换为calendar的logger路径
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/cache" // 替换为calendar的cache路径
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/dao"   // 替换为calendar的dao路径
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/model"
)

// 定义接口
type CalendarService interface {
	GetCalendar(ctx context.Context, year int64) (*domain.Calendar, error)
	SaveCalendar(ctx context.Context, calendar *domain.Calendar) error
	DelCalendar(ctx context.Context, year int64) error
}

// 定义错误结构体
var (
	GET_CALENDAR_ERROR = func(err error) error {
		return errorx.New(calendarv1.ErrorGetCalendarError("获取calendar失败"), "dao", err)
	}

	DEL_CALENDAR_ERROR = func(err error) error {
		return errorx.New(calendarv1.ErrorDelCalendarError("删除calendar失败"), "dao", err)
	}

	SAVE_CALENDAR_ERROR = func(err error) error {
		return errorx.New(calendarv1.ErrorSaveCalendarError("删除calendar失败"), "dao", err)
	}
)

// 缓存的CalendarRepository实现
type CachedCalendarService struct {
	dao   dao.CalendarDAO
	cache cache.CalendarCache
	l     logger.Logger
}

// 构造函数，返回缓存的CalendarRepository
func NewCachedCalendarService(dao dao.CalendarDAO, cache cache.CalendarCache, l logger.Logger) CalendarService {
	return &CachedCalendarService{dao: dao, cache: cache, l: l}
}

// 从缓存或数据库获取日历
func (s *CachedCalendarService) GetCalendar(ctx context.Context, year int64) (*domain.Calendar, error) {
	// 尝试从缓存获取
	res, err := s.cache.GetCalendar(ctx, year)
	if err == nil {
		return res, nil
	}
	s.l.Info("从缓存获取失败", logger.FormatLog("cache", err)...)

	// 如果缓存中不存在则从数据库获取
	calendar, err := s.dao.GetCalendar(ctx, year)
	if err != nil {
		return &domain.Calendar{}, GET_CALENDAR_ERROR(err)
	}

	// 异步写入缓存，牺牲一定的一致性
	go func() {
		ctx = context.Background()
		err = s.cache.SetCalendar(ctx, &domain.Calendar{Year: calendar.Year, Link: calendar.Link}, calendar.Year)
		if err != nil {
			s.l.Error("回写资源失败", logger.FormatLog("cache", err)...)
		}
	}()

	//赋值给res
	res.Year = calendar.Year
	res.Link = calendar.Link

	return res, nil
}

// 保存日历信息并更新缓存
func (s *CachedCalendarService) SaveCalendar(ctx context.Context, calendar *domain.Calendar) error {
	//此处无视错误,如果出错就等于存一个新的值似乎不是很优秀?可能会造成一致性问题
	c, err := s.dao.GetCalendar(ctx, calendar.Year)
	if err != nil {
		c = &model.Calendar{}
	}
	//更新
	c.Year = calendar.Year
	c.Link = calendar.Link
	// 保存到数据库
	err = s.dao.SaveCalendar(ctx, c)
	if err != nil {
		return SAVE_CALENDAR_ERROR(err)
	}

	// 异步写入缓存，牺牲一定的一致性
	go func() {
		ctx = context.Background()
		err = s.cache.SetCalendar(ctx, &domain.Calendar{Year: calendar.Year, Link: calendar.Link}, calendar.Year)
		if err != nil {
			s.l.Error("回写资源失败", logger.FormatLog("cache", err)...)
		}

	}()

	return nil
}

// 删除日历并更新缓存
func (s *CachedCalendarService) DelCalendar(ctx context.Context, year int64) error {
	//删除数据库中资源
	_, err := s.dao.DelCalendar(ctx, year)
	if err != nil {
		return DEL_CALENDAR_ERROR(err)
	}

	//异步删除指定缓存资源
	go func() {
		ctx = context.Background()
		err := s.cache.ClearCalendarCache(ctx, year)
		if err != nil {
			s.l.Error("清除指定缓存失败", logger.FormatLog("cache", err)...)
		}
	}()
	return nil
}
