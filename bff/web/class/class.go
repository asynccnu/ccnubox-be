package class

import (
	"time"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/gin-gonic/gin"
)

type ClassHandler struct {
	ClassListClient classlistv1.ClasserClient
	Administrators  map[string]struct{} // 这里注入的是管理员权限验证配置
	l               logger.Logger
}

func NewClassListHandler(
	ClassListClient classlistv1.ClasserClient,
	administrators map[string]struct{},
	l logger.Logger,
) *ClassHandler {
	return &ClassHandler{
		ClassListClient: ClassListClient,
		Administrators:  administrators,
		l:               l,
	}
}

func (c *ClassHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/class")
	sg.GET("/get", authMiddleware, ginx.WrapClaimsAndReq(c.GetClassList))
	sg.POST("/add", authMiddleware, ginx.WrapClaimsAndReq(c.AddClass))
	sg.POST("/delete", authMiddleware, ginx.WrapClaimsAndReq(c.DeleteClass))
	sg.PUT("/update", authMiddleware, ginx.WrapClaimsAndReq(c.UpdateClass))
	sg.GET("/day/get", ginx.Wrap(c.GetSchoolDay))
	sg.POST("/note/insert", authMiddleware, ginx.WrapClaimsAndReq(c.InsertClassNote))
	sg.POST("/note/delete", authMiddleware, ginx.WrapClaimsAndReq(c.DeleteClassNote))
}

// GetClassList 获取课表
// @Summary 获取课表
// @Description 根据学年、学期获取当前登录学生的课表。refresh=false 优先读缓存/本地数据，refresh=true 会触发刷新。成功时 code=0；业务失败通常仍由统一响应体返回 code=50001 和 msg。
// @Tags class
// @Produce json
// @Param Authorization header string true "Bearer Token，例如 Bearer xxx"
// @Param request query GetClassListRequest true "获取课表请求参数"
// @Success 200 {object} web.Response{data=GetClassListResp} "成功返回课表"
// @Failure 401 {object} web.Response "未登录或 token 无效，code=40001"
// @Failure 422 {object} web.Response "请求参数错误，code=40002"
// @Router /class/get [get]
func (c *ClassHandler) GetClassList(ctx *gin.Context, req GetClassListRequest, uc ijwt.UserClaims) (web.Response, error) {
	if req.Refresh == nil {
		req.Refresh = new(bool)
		*req.Refresh = false
	}
	getResp, err := c.ClassListClient.GetClass(ctx, &classlistv1.GetClassRequest{
		StuId:    uc.StudentId,
		Semester: req.Semester,
		Year:     req.Year,
		Refresh:  *req.Refresh,
	})
	if err != nil {
		return web.Response{}, errs.GET_CLASS_LIST_ERROR(err)
	}

	respClasses := make([]*ClassInfo, 0, len(getResp.Classes))

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
			Note:         class.Info.Note,
			IsOfficial:   class.Info.IsOfficial,
			Nature:       class.Info.Nature,
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
// @Summary 添加自定义课程
// @Description 给当前登录学生添加一门自定义课程。weeks 必须是周次数组，例如 [1,2,3]；dur_class 是节次范围，例如 "1-2"。成功时 code=0；添加失败、课程已存在、时间冲突等会返回 code=50001 和对应 msg。
// @Tags class
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token，例如 Bearer xxx"
// @Param request body AddClassRequest true "自定义课程信息"
// @Success 200 {object} web.Response "成功添加课程，code=0"
// @Failure 401 {object} web.Response "未登录或 token 无效，code=40001"
// @Failure 422 {object} web.Response "请求参数错误，code=40002"
// @Router /class/add [post]
func (c *ClassHandler) AddClass(ctx *gin.Context, req AddClassRequest, uc ijwt.UserClaims) (web.Response, error) {
	weeks := convertWeekFromArrayToInt(req.Weeks)

	preq := &classlistv1.AddClassRequest{
		StuId:    uc.StudentId,
		Name:     req.Name,
		DurClass: req.DurClass,
		Where:    req.Where,
		Teacher:  req.Teacher,
		Weeks:    weeks,
		Semester: req.Semester,
		Year:     req.Year,
		Day:      req.Day,
		Credit:   req.Credit,
	}

	_, err := c.ClassListClient.AddClass(ctx, preq)
	if err != nil {
		return web.Response{}, errs.ADD_CLASS_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// DeleteClass 删除课表
// @Summary 删除自定义课程
// @Description 根据课程 ID 删除当前登录学生的自定义课程。教务系统导入课程不支持删除。成功时 code=0；删除失败或删除官方课程会返回 code=50001 和 msg。
// @Tags class
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token，例如 Bearer xxx"
// @Param request body DeleteClassRequest true "删除课程请求"
// @Success 200 {object} web.Response "成功删除课程，code=0"
// @Failure 401 {object} web.Response "未登录或 token 无效，code=40001"
// @Failure 422 {object} web.Response "请求参数错误，code=40002"
// @Router /class/delete [post]
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
// @Summary 更新自定义课程
// @Description 根据课程 ID 更新当前登录学生的自定义课程。可更新课程名称、节次、地点、教师、周次、星期几、学分；更新后课程 ID 可能改变。成功时 code=0；更新失败或时间冲突会返回 code=50001 和 msg。
// @Tags class
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token，例如 Bearer xxx"
// @Param request body UpdateClassRequest true "更新课程请求"
// @Success 200 {object} web.Response "成功更新课程，code=0"
// @Failure 401 {object} web.Response "未登录或 token 无效，code=40001"
// @Failure 422 {object} web.Response "请求参数错误，code=40002"
// @Router /class/update [put]
func (c *ClassHandler) UpdateClass(ctx *gin.Context, req UpdateClassRequest, uc ijwt.UserClaims) (web.Response, error) {
	var weeks *int64
	if len(req.Weeks) > 0 {
		tmpWeeks := convertWeekFromArrayToInt(req.Weeks)
		weeks = &tmpWeeks
	}

	preq := &classlistv1.UpdateClassRequest{
		ClassId:  req.ClassId,
		StuId:    uc.StudentId,
		Name:     req.Name,
		DurClass: req.DurClass,
		Where:    req.Where,
		Teacher:  req.Teacher,
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

// GetSchoolDay 获取当前周
// @Summary 获取学期日期配置
// @Description 获取当前学期的开学日期和放假日期，返回秒级时间戳。前端用 school_time 计算当前周，用 holiday_time 判断学期边界。成功时 code=0；配置缺失或格式错误会返回 code=50001。
// @Tags class
// @Produce json
// @Success 200 {object} web.Response{data=GetSchoolDayResp} "成功返回学期日期配置"
// @Router /class/day/get [get]
func (c *ClassHandler) GetSchoolDay(ctx *gin.Context) (web.Response, error) {
	res, err := c.ClassListClient.GetSchoolDay(ctx, &classlistv1.GetSchoolDayReq{})
	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, errs.TYPE_CHANGE_ERROR(err)
	}
	// 加载 "Asia/Shanghai" 时区
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

// InsertClassNote 插入课程备注
// @Summary 添加或更新课程备注
// @Description 根据课程 ID 给当前登录学生的课程添加或更新备注。成功时 code=0；课程不存在或保存失败会返回 code=50001 和 msg。
// @Tags class
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token，例如 Bearer xxx"
// @Param request body UpdateClassNoteReq true "添加或更新课程备注请求"
// @Success 200 {object} web.Response "成功添加或更新课程备注，code=0"
// @Failure 401 {object} web.Response "未登录或 token 无效，code=40001"
// @Failure 422 {object} web.Response "请求参数错误，code=40002"
// @Router /class/note/insert [post]
func (c *ClassHandler) InsertClassNote(ctx *gin.Context, req UpdateClassNoteReq, uc ijwt.UserClaims) (web.Response, error) {
	resp, err := c.ClassListClient.UpdateClassNote(ctx, &classlistv1.UpdateClassNoteReq{
		StuId:    uc.StudentId,
		Year:     req.Year,
		Semester: req.Semester,
		ClassId:  req.ClassId,
		Note:     req.Note,
	})
	if err != nil {
		return web.Response{}, errs.UPDATE_CLASS_ERROR(err)
	}
	return web.Response{
		Msg: resp.Msg,
	}, nil
}

// DeleteClassNote 删除课程备注
// @Summary 删除课程备注
// @Description 根据课程 ID 删除当前登录学生的课程备注。成功时 code=0；课程不存在或删除失败会返回 code=50001 和 msg。
// @Tags class
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token，例如 Bearer xxx"
// @Param request body DeleteClassNoteReq true "删除课程备注请求"
// @Success 200 {object} web.Response "成功删除课程备注，code=0"
// @Failure 401 {object} web.Response "未登录或 token 无效，code=40001"
// @Failure 422 {object} web.Response "请求参数错误，code=40002"
// @Router /class/note/delete [post]
func (c *ClassHandler) DeleteClassNote(ctx *gin.Context, req DeleteClassNoteReq, uc ijwt.UserClaims) (web.Response, error) {
	resp, err := c.ClassListClient.DeleteClassNote(ctx, &classlistv1.DeleteClassNoteReq{
		StuId:    uc.StudentId,
		Year:     req.Year,
		Semester: req.Semester,
		ClassId:  req.ClassId,
	})
	if err != nil {
		return web.Response{}, errs.UPDATE_CLASS_ERROR(err)
	}
	return web.Response{
		Msg: resp.Msg,
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
