package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-calendar/domain"           // 替换为calendar的domain路径
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/cache" // 替换为calendar的cache路径
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/dao"   // 替换为calendar的dao路径
	"github.com/asynccnu/ccnubox-be/be-calendar/repository/model"
	calendarv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/calendar/v1"
	errorx "github.com/asynccnu/ccnubox-be/common/pkg/errorx/rpcerr"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger" // 替换为calendar的logger路径
)

// 定义接口
type CalendarService interface {
	GetCalendars(ctx context.Context) ([]domain.Calendar, error)
	SaveCalendar(ctx context.Context, calendar *domain.Calendar) error
	DelCalendar(ctx context.Context, year int64) error
	GetCalendar(ctx context.Context, year int64) (*domain.Calendar, error)
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
func (s *CachedCalendarService) GetCalendars(ctx context.Context) ([]domain.Calendar, error) {
	// 尝试从缓存获取
	res, err := s.cache.GetCalendars(ctx)
	if err == nil {
		return res, nil
	}
	s.l.Info("从缓存获取失败", logger.Error(err))

	// 如果缓存中不存在则从数据库获取
	calendars, err := s.dao.GetCalendars(ctx)
	if err != nil {
		return []domain.Calendar{}, GET_CALENDAR_ERROR(err)
	}

	res = convModelsToDomains(calendars)

	// 异步写入缓存，牺牲一定的一致性
	go func() {
		ctx = context.Background()
		err = s.cache.SetCalendar(ctx, res)
		if err != nil {
			s.l.Error("回写资源失败", logger.Error(err))
		}
	}()

	return res, nil
}

func (s *CachedCalendarService) GetCalendar(ctx context.Context, year int64) (*domain.Calendar, error) {

	// 如果缓存中不存在则从数据库获取
	calendar, err := s.dao.GetCalendar(ctx, year)
	if err != nil {
		return &domain.Calendar{}, GET_CALENDAR_ERROR(err)
	}

	return &domain.Calendar{
		Year: calendar.Year,
		Link: calendar.Link,
	}, nil
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
		c, err := s.dao.GetCalendars(ctx)
		if err != nil {
			s.l.Error("获取日历资源失败", logger.Error(err))
			return
		}

		err = s.cache.SetCalendar(ctx, convModelsToDomains(c))
		if err != nil {
			s.l.Error("回写资源失败", logger.Error(err))
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
		c, err := s.dao.GetCalendars(ctx)
		if err != nil {
			s.l.Error("获取日历资源失败", logger.Error(err))
			return
		}

		err = s.cache.SetCalendar(ctx, convModelsToDomains(c))
		if err != nil {
			s.l.Error("回写资源失败", logger.Error(err))
		}
	}()
	return nil
}

func convModelsToDomains(calendars []model.Calendar) []domain.Calendar {
	res := make([]domain.Calendar, 0, len(calendars))
	for _, c := range calendars {
		res = append(res, domain.Calendar{
			Year: c.Year,
			Link: c.Link,
		})
	}
	return res
}

func convDomainsToModels(calendars []domain.Calendar) []model.Calendar {
	res := make([]model.Calendar, 0, len(calendars))
	for _, c := range calendars {
		res = append(res, model.Calendar{
			Year: c.Year,
			Link: c.Link,
		})
	}
	return res
}
