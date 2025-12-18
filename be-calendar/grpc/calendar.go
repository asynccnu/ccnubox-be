package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-calendar/domain"
	"github.com/asynccnu/ccnubox-be/be-calendar/service"                         // 替换为calendar的服务路径
	calendarv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/calendar/v1" // 替换为calendar的proto包路径
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type CalendarServiceServer struct {
	calendarv1.UnimplementedCalendarServiceServer
	svc service.CalendarService
}

func NewCalendarServiceServer(svc service.CalendarService) *CalendarServiceServer {
	return &CalendarServiceServer{svc: svc}
}

func (c *CalendarServiceServer) GetCalendars(ctx context.Context, request *calendarv1.GetCalendarsRequest) (*calendarv1.GetCalendarsResponse, error) {
	calendars, err := c.svc.GetCalendars(ctx)
	if err != nil {
		return nil, err
	}
	return &calendarv1.GetCalendarsResponse{
		Calendars: convDomainsToGRPC(calendars),
	}, nil
}

func (c *CalendarServiceServer) SaveCalendar(ctx context.Context, request *calendarv1.SaveCalendarRequest) (*calendarv1.SaveCalendarResponse, error) {
	err := c.svc.SaveCalendar(ctx, &domain.Calendar{
		Year: request.Calendar.GetYear(),
		Link: request.Calendar.GetLink(),
	})
	if err != nil {
		return nil, err
	}
	return &calendarv1.SaveCalendarResponse{}, nil
}

func (c *CalendarServiceServer) DelCalendar(ctx context.Context, request *calendarv1.DelCalendarRequest) (*calendarv1.DelCalendarResponse, error) {
	err := c.svc.DelCalendar(ctx, request.GetYear())
	if err != nil {
		return nil, err
	}
	return &calendarv1.DelCalendarResponse{}, nil
}

// 注册为grpc服务
func (c *CalendarServiceServer) Register(server *grpc.Server) {
	calendarv1.RegisterCalendarServiceServer(server, c)
}
func convDomainsToGRPC(calendars []domain.Calendar) []*calendarv1.Calendar {
	res := make([]*calendarv1.Calendar, 0, len(calendars))
	for _, c := range calendars {
		res = append(res, &calendarv1.Calendar{
			Year: c.Year,
			Link: c.Link,
		})
	}
	return res
}
