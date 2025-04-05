package grpc

import (
	"context"
	calendarv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/calendar/v1" // 替换为calendar的proto包路径
	"github.com/asynccnu/ccnubox-be/be-calendar/domain"
	"github.com/asynccnu/ccnubox-be/be-calendar/service" // 替换为calendar的服务路径
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type CalendarServiceServer struct {
	calendarv1.UnimplementedCalendarServiceServer
	svc service.CalendarService
}

func NewCalendarServiceServer(svc service.CalendarService) *CalendarServiceServer {
	return &CalendarServiceServer{svc: svc}
}

func (c *CalendarServiceServer) GetCalendar(ctx context.Context, request *calendarv1.GetCalendarRequest) (*calendarv1.GetCalendarResponse, error) {
	calendar, err := c.svc.GetCalendar(ctx, request.GetYear())
	if err != nil {
		return nil, err
	}
	return &calendarv1.GetCalendarResponse{
		Link: calendar.Link,
		Year: calendar.Year,
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
