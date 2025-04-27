package class

import (
	"errors"
	cs "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classService/v1"
	classlistv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"time"
)

type ClassHandler struct {
	ClassListClient    classlistv1.ClasserClient
	ClassServiceClinet cs.ClassServiceClient
	Administrators     map[string]struct{} //这里注入的是管理员权限验证配置
}

func NewClassListHandler(
	ClassListClient classlistv1.ClasserClient,
	ClassServiceClinet cs.ClassServiceClient,
	administrators map[string]struct{}) *ClassHandler {
	return &ClassHandler{
		ClassListClient:    ClassListClient,
		ClassServiceClinet: ClassServiceClinet,
		Administrators:     administrators,
	}
}

func (c *ClassHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/class")
	sg.GET("/get", authMiddleware, ginx.WrapClaimsAndReq(c.GetClassList))
	sg.POST("/add", authMiddleware, ginx.WrapClaimsAndReq(c.AddClass))
	sg.DELETE("/delete", authMiddleware, ginx.WrapClaimsAndReq(c.DeleteClass))
	sg.PUT("/update", authMiddleware, ginx.WrapClaimsAndReq(c.UpdateClass))
	sg.GET("/getRecycle", authMiddleware, ginx.WrapClaimsAndReq(c.GetRecycleBinClassInfos))
	sg.PUT("/recover", authMiddleware, ginx.WrapClaimsAndReq(c.RecoverClass))
	sg.GET("/search", authMiddleware, ginx.WrapReq(c.SearchClass))
	sg.GET("/day/get", ginx.Wrap(c.GetSchoolDay))
}

// GetClassList 获取课表
// @Summary 获取课表
// @Description 根据学期、学年等条件获取课表
// @Tags 课表
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request query GetClassListRequest true "获取课表请求参数"
// @Success 200 {object} web.Response{data=GetClassListResp} "成功返回课表"
// @Router /class/get [get]
func (c *ClassHandler) GetClassList(ctx *gin.Context, req GetClassListRequest, uc ijwt.UserClaims) (web.Response, error) {

	getResp, err := c.ClassListClient.GetClass(ctx, &classlistv1.GetClassRequest{
		StuId:    uc.StudentId,
		Semester: req.Semester,
		Year:     req.Year,
		Refresh:  req.Refresh,
	})
	if err != nil {
		return web.Response{}, errs.GET_CLASS_LIST_ERROR(err)
	}

	var respClasses = make([]*ClassInfo, 0, len(getResp.Classes))

	for _, class := range getResp.Classes {
		respClasses = append(respClasses, &ClassInfo{
			ID:           class.Info.Id,
			Day:          class.Info.Day,
			Teacher:      class.Info.Teacher,
			Where:        class.Info.Where,
			ClassWhen:    class.Info.ClassWhen,
			WeekDuration: class.Info.WeekDuration,
			Classname:    class.Info.Classname,
			Credit:       class.Info.Credit,
			Weeks:        convertWeekFromIntToArray(class.Info.Weeks),
			Semester:     class.Info.Semester,
			Year:         class.Info.Year,
		})
	}

	resp := GetClassListResp{
		Classes:         respClasses,
		LastRefreshTime: getResp.LastTime,
	}

	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// AddClass 添加课表
// @Summary 添加课表
// @Description 添加新的课表
// @Tags 课表
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body AddClassRequest true "课表信息"
// @Success 200 {object} web.Response "成功添加课表"
// @Router /class/add [post]
func (c *ClassHandler) AddClass(ctx *gin.Context, req AddClassRequest, uc ijwt.UserClaims) (web.Response, error) {

	weeks := convertWeekFromArrayToInt(req.Weeks)

	var preq = &cs.AddClassRequest{
		StuId:    uc.StudentId,
		Name:     req.Name,
		DurClass: req.DurClass,
		Where:    req.Where,
		Teacher:  req.Where,
		Weeks:    weeks,
		Semester: req.Semester,
		Year:     req.Year,
		Day:      req.Day,
		Credit:   req.Credit,
	}

	_, err := c.ClassServiceClinet.AddClass(ctx, preq)
	if err != nil {
		return web.Response{}, errs.ADD_CLASS_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// DeleteClass 删除课表
// @Summary 删除课表
// @Description 根据课表ID删除课表
// @Tags 课表
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body DeleteClassRequest true "删除课表请求"
// @Success 200 {object} web.Response "成功删除课表"
// @Router /class/delete [delete]
func (c *ClassHandler) DeleteClass(ctx *gin.Context, req DeleteClassRequest, uc ijwt.UserClaims) (web.Response, error) {
	_, err := c.ClassListClient.DeleteClass(ctx, &classlistv1.DeleteClassRequest{
		Id:       req.Id,
		StuId:    uc.StudentId,
		Year:     req.Year,
		Semester: req.Semester,
	})
	if err != nil {
		return web.Response{}, errs.DELETE_CLASS_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// UpdateClass 更新课表信息
// @Summary 更新课表信息
// @Description 根据课表ID更新课表信息
// @Tags 课表
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body UpdateClassRequest true "更新课表请求"
// @Success 200 {object} web.Response "成功更新课表"
// @Router /class/update [put]
func (c *ClassHandler) UpdateClass(ctx *gin.Context, req UpdateClassRequest, uc ijwt.UserClaims) (web.Response, error) {
	var weeks *int64
	if len(req.Weeks) > 0 {
		tmpWeeks := convertWeekFromArrayToInt(req.Weeks)
		weeks = &tmpWeeks
	}

	var preq = &classlistv1.UpdateClassRequest{
		ClassId:  req.ClassId,
		StuId:    uc.StudentId,
		Name:     req.Name,
		DurClass: req.DurClass,
		Where:    req.Where,
		Teacher:  req.Where,
		Weeks:    weeks,
		Semester: req.Semester,
		Year:     req.Year,
		Day:      req.Day,
		Credit:   req.Credit,
	}

	_, err := c.ClassListClient.UpdateClass(ctx, preq)
	if err != nil {
		return web.Response{}, errs.UPDATE_CLASS_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// GetRecycleBinClassInfos 获取回收站中的课表信息
// @Summary 获取回收站课表信息
// @Description 获取已删除但未彻底清除的课表信息
// @Tags 课表
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request query GetRecycleBinClassInfosReq true "获取回收站中的课表信息参数"
// @Success 200 {object} web.Response{data=GetRecycleBinClassInfosResp} "成功获取回收站课表信息"
// @Router /class/getRecycle [get]
func (c *ClassHandler) GetRecycleBinClassInfos(ctx *gin.Context, req GetRecycleBinClassInfosReq, uc ijwt.UserClaims) (web.Response, error) {
	classes, err := c.ClassListClient.GetRecycleBinClassInfos(ctx, &classlistv1.GetRecycleBinClassRequest{
		StuId:    uc.StudentId,
		Year:     req.Year,
		Semester: req.Semester,
	})
	if err != nil {
		return web.Response{}, errs.GET_RECYCLE_CLASS_ERROR(err)
	}
	var resp GetRecycleBinClassInfosResp
	err = copier.Copy(&resp.ClassInfos, &classes.ClassInfos)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}
	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// RecoverClass 恢复课表
// @Summary 恢复课表
// @Description 从回收站恢复课表
// @Tags 课表
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body RecoverClassRequest true "恢复课表请求"
// @Success 200 {object} web.Response "成功恢复课表"
// @Router /class/recover [put]
func (c *ClassHandler) RecoverClass(ctx *gin.Context, req RecoverClassRequest, uc ijwt.UserClaims) (web.Response, error) {
	_, err := c.ClassListClient.RecoverClass(ctx, &classlistv1.RecoverClassRequest{
		StuId:    uc.StudentId,
		Year:     req.Year,
		Semester: req.Semester,
		ClassId:  req.ClassId,
	})
	if err != nil {
		return web.Response{}, errs.RECOVER_CLASS_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// SearchClass 查询课程
// @Summary 搜索课程
// @Description 根据关键词[教师或者课程名]搜索课程,**注意,但当返回的结果数量大于page_size时,代表还有下一页**,最开始请求的是第一页
// @Tags 课表
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request query SearchRequest true "查询课程请求参数"
// @Success 200 {object} web.Response{data=SearchClassResp} "成功搜索到课程"
// @Router /class/search [get]
func (c *ClassHandler) SearchClass(ctx *gin.Context, req SearchRequest) (web.Response, error) {
	if req.Page <= 0 || req.PageSize <= 0 {
		return web.Response{}, errs.INVALID_PARAM_VALUE_ERROR(errors.New("page or pageSize must be greater than 0"))
	}

	classes, err := c.ClassServiceClinet.SearchClass(ctx, &cs.SearchRequest{
		Year:           req.Year,
		Semester:       req.Semester,
		SearchKeyWords: req.SearchKeyWords,
		Page:           int32(req.Page),
		PageSize:       int32(req.PageSize),
	})

	if err != nil {
		return web.Response{}, errs.SEARCH_CLASS_ERROR(err)
	}
	var resp SearchClassResp

	respClasses := make([]*ClassInfo, 0, len(classes.ClassInfos))

	for _, class := range classes.ClassInfos {
		respClasses = append(respClasses, &ClassInfo{
			ID:           class.Id,
			Day:          class.Day,
			Teacher:      class.Teacher,
			Where:        class.Where,
			ClassWhen:    class.ClassWhen,
			WeekDuration: class.WeekDuration,
			Classname:    class.Classname,
			Credit:       class.Credit,
			Weeks:        convertWeekFromIntToArray(class.Weeks),
			Semester:     class.Semester,
			Year:         class.Year,
		})
	}

	resp.ClassInfos = respClasses

	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// GetSchoolDay 获取当前周
// @Summary 获取当前周
// @Description 获取当前周
// @Tags 课表
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} web.Response{data=GetSchoolDayResp} "成功获取到当前周"
// @Router /class/day/get [get]
func (c *ClassHandler) GetSchoolDay(ctx *gin.Context) (web.Response, error) {

	res, err := c.ClassListClient.GetSchoolDay(ctx, &classlistv1.GetSchoolDayReq{})
	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, errs.TYPE_CHANGE_ERROR(err)
	}
	//加载 "Asia/Shanghai" 时区
	loc, _ := time.LoadLocation("Asia/Shanghai")
	holiday, err := time.ParseInLocation("2006-01-02", res.GetHolidayTime(), loc)
	if err != nil {
		return web.Response{}, nil
	}

	school, err := time.ParseInLocation("2006-01-02", res.GetSchoolTime(), loc)
	if err != nil {
		return web.Response{}, nil
	}

	return web.Response{
		Msg: "Success",
		Data: GetSchoolDayResp{

			HolidayTime: holiday.Unix(),
			SchoolTime:  school.Unix(),
		},
	}, nil
}

func convertWeekFromArrayToInt(weeks []int) int64 {
	var res int64

	for _, week := range weeks {
		if week < 1 || week >= 30 {
			continue
		}

		res |= 1 << (week - 1)
	}
	return res
}

func convertWeekFromIntToArray(weeks int64) []int {
	var res []int

	for i := 0; i < 30; i++ {
		if (weeks & (1 << uint(i))) != 0 {
			res = append(res, i+1)
		}
	}
	return res
}
