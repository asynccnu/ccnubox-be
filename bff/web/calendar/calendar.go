package calendar

import (
	"fmt"
	calendarv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/calendar/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

type CalendarHandler struct {
	calendarClient calendarv1.CalendarServiceClient
	Administrators map[string]struct{}
}

func NewCalendarHandler(calendarClient calendarv1.CalendarServiceClient,
	administrators map[string]struct{}) *CalendarHandler {
	return &CalendarHandler{calendarClient: calendarClient, Administrators: administrators}
}

func (h *CalendarHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/calendar")
	sg.GET("/getCalendars", ginx.Wrap(h.GetCalendars))
	sg.POST("/saveCalendar", authMiddleware, ginx.WrapClaimsAndReq(h.SaveCalendar))
	sg.POST("/delCalendar", authMiddleware, ginx.WrapClaimsAndReq(h.DelCalendar))
}

// GetCalendar  获取日历列表
// @Summary 获取日历列表
// @Description 获取日历列表
// @Tags calendar
// @Accept  json
// @Produce  json
// @Success 200 {object} web.Response{data=GetCalendarsResponse} "成功"
// @Router /calendar/getCalendars [get]
func (h *CalendarHandler) GetCalendars(ctx *gin.Context) (web.Response, error) {
	calendar, err := h.calendarClient.GetCalendars(ctx, &calendarv1.GetCalendarsRequest{})
	if err != nil {
		return web.Response{}, errs.GET_CALENDAR_ERROR(err)
	}

	//类型转换
	var resp GetCalendarsResponse
	err = copier.Copy(&resp, &calendar)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}
	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// SaveCalendar 保存日历内容
// @Summary 保存日历内容
// @Description 保存日历内容
// @Tags calendar
// @Accept json
// @Produce json
// @Param request body SaveCalendarRequest true "保存日历内容请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /calendar/saveCalendar [post]
func (h *CalendarHandler) SaveCalendar(ctx *gin.Context, req SaveCalendarRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.calendarClient.SaveCalendar(ctx, &calendarv1.SaveCalendarRequest{Calendar: &calendarv1.Calendar{
		Link: req.Link,
		Year: req.Year,
	}})

	if err != nil {
		return web.Response{}, errs.Save_CALENDAR_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

// DelCalendar 删除日历内容
// @Summary 删除日历内容
// @Description 删除日历内容
// @Tags calendar
// @Accept json
// @Produce json
// @Param request body DelCalendarRequest true "删除日历内容请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /calendar/delCalendar [post]
func (h *CalendarHandler) DelCalendar(ctx *gin.Context, req DelCalendarRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.calendarClient.DelCalendar(ctx, &calendarv1.DelCalendarRequest{Year: req.Year})
	if err != nil {
		return web.Response{}, errs.Del_CALENDAR_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

func (h *CalendarHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
