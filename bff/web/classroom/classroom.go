package classroom

import (
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	cs "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classService/v1"
	"github.com/gin-gonic/gin"
)

type ClassRoomHandler struct {
	ClassRoomClient cs.FreeClassroomSvcClient
}

func NewClassRoomHandler(ClassRoomClient cs.FreeClassroomSvcClient) *ClassRoomHandler {
	return &ClassRoomHandler{
		ClassRoomClient: ClassRoomClient,
	}
}

func (c *ClassRoomHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/classroom")
	sg.GET("/getFreeClassRoom", authMiddleware, ginx.WrapClaimsAndReq(c.GetFreeClassRoom))
	sg.GET("/list", ginx.Wrap(c.GetClassrooms))
}

// GetFreeClassRoom 查询空闲教室
// @Summary 查询空闲教室
// @Description 根据学年、学期、周次、节次、地点等信息查询空闲教室列表
// @Tags classroom
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param year query string true "学年，如：2024-2025"
// @Param semester query string true "学期，如：1 或 2"
// @Param week query int true "第几周"
// @Param day query int true "星期几，1-7"
// @Param sections query []int true "第几节课（可多选）"
// @Param wherePrefix query string true "地点前缀，如 n1 表示南湖一楼"
// @Success 200 {object} web.Response{data=GetFreeClassRoomResp} "查询成功"
// @Router /classroom/getFreeClassRoom [get]
func (c *ClassRoomHandler) GetFreeClassRoom(ctx *gin.Context, req GetFreeClassRoomReq, uc ijwt.UserClaims) (web.Response, error) {
	resp, err := c.ClassRoomClient.QueryFreeClassroom(ctx, &cs.QueryFreeClassroomReq{
		Year:        req.Year,
		Semester:    req.Semester,
		Week:        req.Week,
		Day:         req.Day,
		Sections:    req.Sections,
		WherePrefix: req.WherePrefix,
		StuID:       uc.StudentId,
	})
	if err != nil {
		return web.Response{}, err
	}

	// 你可以根据实际返回内容加工成 web.Response
	return web.Response{
		Code: 0,
		Msg:  "查询成功",
		Data: convertToGetFreeClassRoomResp(resp),
	}, nil
}

// GetClassrooms returns the classroom list from be-class.
// @Summary 获取教室列表
// @Description 返回 be-class 中 classrooms.json 的教室列表
// @Tags classroom
// @Produce json
// @Success 200 {object} web.Response{data=GetClassroomsResp} "查询成功"
// @Router /classroom/list [get]
func (c *ClassRoomHandler) GetClassrooms(ctx *gin.Context) (web.Response, error) {
	resp, err := c.ClassRoomClient.GetClassrooms(ctx, &cs.GetClassroomsReq{})
	if err != nil {
		return web.Response{}, err
	}

	return web.Response{
		Code: 0,
		Msg:  "查询成功",
		Data: convertToGetClassroomsResp(resp),
	}, nil
}
