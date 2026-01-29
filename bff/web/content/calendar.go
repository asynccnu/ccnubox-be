package content

import (
	"fmt"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

// RegisterCalendarRoute 注册 Calendar 相关路由
func (h *ContentHandler) RegisterCalendarRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/calendar")
	// 注意：GetCalendars 通常不需要 authMiddleware，除非你希望统计学生 ID
	sg.GET("/getCalendars", ginx.Wrap(h.GetCalendars))
	sg.POST("/saveCalendar", authMiddleware, ginx.WrapClaimsAndReq(h.SaveCalendar))
	sg.POST("/delCalendar", authMiddleware, ginx.WrapClaimsAndReq(h.DelCalendar))
}

// GetCalendars 获取日历列表
// @Summary 获取日历列表
// @Description 获取日历列表
// @Tags calendar
// @Success 200 {object} web.Response{data=GetCalendarsResponse} "成功"
// @Router /calendar/getCalendars [get]
func (h *ContentHandler) GetCalendars(ctx *gin.Context) (web.Response, error) {
	// 统一调用 contentClient
	resp, err := h.contentClient.GetCalendars(ctx, &contentv1.GetCalendarsRequest{})
	if err != nil {
		return web.Response{}, errs.GET_CALENDAR_ERROR(err)
	}

	// 类型转换
	var data GetCalendarsResponse
	err = copier.Copy(&data.Calendars, &resp.Calendars)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: data,
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
func (h *ContentHandler) SaveCalendar(ctx *gin.Context, req SaveCalendarRequest, uc ijwt.UserClaims) (web.Response, error) {
	// 权限校验
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	// 统一使用 contentClient.SaveCalendar
	_, err := h.contentClient.SaveCalendar(ctx, &contentv1.SaveCalendarRequest{
		Calendar: &contentv1.Calendar{
			Year: req.Year,
			Link: req.Link,
		},
	})
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
func (h *ContentHandler) DelCalendar(ctx *gin.Context, req DelCalendarRequest, uc ijwt.UserClaims) (web.Response, error) {
	// 权限校验
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	// 调用 contentClient
	_, err := h.contentClient.DelCalendar(ctx, &contentv1.DelCalendarRequest{
		Year: req.Year,
	})
	if err != nil {
		return web.Response{}, errs.Del_CALENDAR_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}
