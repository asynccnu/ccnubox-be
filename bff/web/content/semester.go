package content

import (
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/gin-gonic/gin"
)

func (h *ContentHandler) RegisterSemesterRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/semester")
	sg.GET("/getSemester", authMiddleware, ginx.Wrap(h.GetSemester))
	sg.POST("/saveSemester", authMiddleware, ginx.WrapClaimsAndReq(h.SaveSemester))
	sg.GET("/getSemesterList", authMiddleware, ginx.WrapClaims(h.GetSemesterList))
}

// GetSemester 获取当前所属学期
// @Summary 获取当前所属学期
// @Description 获取当前所属学期
// @Tags semester
// @Success 200 {object} web.Response{data=GetSemesterResponse} "成功"
// @Router /semester/getSemester [get]
func (h *ContentHandler) GetSemester(ctx *gin.Context) (web.Response, error) {
	r := &contentv1.GetSemesterRequest{Date: time.Now().Format("2006-01-02")}
	resp, err := h.contentClient.GetSemester(ctx, r)
	if err != nil {
		return web.Response{}, errs.GET_SEMESTER_ERROR(err)
	}
	if resp.Semester == nil {
		return web.Response{}, errs.GET_SEMESTER_ERROR(errorx.Errorf("获取学期数据为空"))
	}
	return web.Response{
		Msg: "Success",
		Data: Semester{
			Semester:  resp.Semester.Semester,
			StartDate: resp.Semester.StartDate,
			EndDate:   resp.Semester.EndDate,
		},
	}, nil
}

// SaveSemester 保存学期信息
// @Summary 保存学期信息
// @Description 保存学期信息
// @Param request body SaveSemesterRequest true "保存学期信息请求参数"
// @Tags semester
// @Success 200 {object} web.Response{} "成功"
// @Router /semester/saveSemester [post]
func (h *ContentHandler) SaveSemester(ctx *gin.Context, req SaveSemesterRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	r := &contentv1.SaveSemesterRequest{
		Semester: &contentv1.Semester{Semester: req.Semester, StartDate: req.StartDate, EndDate: req.EndDate},
	}

	_, err := h.contentClient.SaveSemester(ctx, r)
	if err != nil {
		return web.Response{}, errs.SAVE_SEMESTER_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// GetSemesterList 获取所有学期信息
// @Summary 获取所有学期信息
// @Description 获取所有学期信息
// @Tags semester
// @Success 200 {object} web.Response{data=GetSemesterListResponse} "成功"
// @Router /semester/getSemesterList [get]
func (h *ContentHandler) GetSemesterList(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	resp, err := h.contentClient.GetSemesterList(ctx, &contentv1.GetSemesterListRequest{StudentId: uc.StudentId})
	if err != nil {
		return web.Response{}, err
	}

	semesters := make([]Semester, 0, len(resp.Semesters))
	for _, s := range resp.Semesters {
		semesters = append(semesters, Semester{
			Semester:  s.GetSemester(),
			StartDate: s.GetStartDate(),
			EndDate:   s.GetEndDate(),
		})
	}

	return web.Response{
		Msg:  "Success",
		Data: semesters,
	}, nil
}
