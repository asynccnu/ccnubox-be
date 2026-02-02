package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

func (c *ContentServiceServer) GetCalendars(ctx context.Context, request *contentv1.GetCalendarsRequest) (*contentv1.GetCalendarsResponse, error) {
	calendars, err := c.svcCalendar.GetList(ctx)
	if err != nil {
		return nil, err
	}
	return &contentv1.GetCalendarsResponse{
		Calendars: calendarDomains2GRPC(calendars),
	}, nil
}

func (c *ContentServiceServer) SaveCalendar(ctx context.Context, request *contentv1.SaveCalendarRequest) (*contentv1.SaveCalendarResponse, error) {
	err := c.svcCalendar.Save(ctx, &domain.Calendar{
		Year: request.Calendar.GetYear(),
		Link: request.Calendar.GetLink(),
	})
	if err != nil {
		return nil, err
	}
	return &contentv1.SaveCalendarResponse{}, nil
}

func (c *ContentServiceServer) DelCalendar(ctx context.Context, request *contentv1.DelCalendarRequest) (*contentv1.DelCalendarResponse, error) {
	err := c.svcCalendar.Del(ctx, request.GetYear())
	if err != nil {
		return nil, err
	}
	return &contentv1.DelCalendarResponse{}, nil
}

func calendarDomains2GRPC(calendars []domain.Calendar) []*contentv1.Calendar {
	res := make([]*contentv1.Calendar, 0, len(calendars))
	for _, c := range calendars {
		res = append(res, &contentv1.Calendar{
			Year: c.Year,
			Link: c.Link,
		})
	}
	return res
}
