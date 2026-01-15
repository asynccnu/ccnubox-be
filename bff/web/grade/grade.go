package grade

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/gin-gonic/gin"
)

type GradeHandler struct {
	GradeClient    gradev1.GradeServiceClient // 注入的是grpc服务
	CounterClient  counterv1.CounterServiceClient
	Administrators map[string]struct{} // 这里注入的是管理员权限验证配置
	l              logger.Logger
}

func NewGradeHandler(
	GradeClient gradev1.GradeServiceClient, // 注入的是grpc服务
	CounterClient counterv1.CounterServiceClient,
	l logger.Logger,
	administrators map[string]struct{},
) *GradeHandler {
	return &GradeHandler{
		GradeClient:    GradeClient,
		CounterClient:  CounterClient,
		Administrators: administrators,
		l:              l,
	}
}

func (h *GradeHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/grade")
	// 这里有三类路由,分别是ginx.WrapClaimsAndReq()有参数且要验证
	sg.POST("/getGradeByTerm", authMiddleware, ginx.WrapClaimsAndReq(h.GetGradeByTerm))
	sg.GET("/getGradeScore", authMiddleware, ginx.WrapClaims(h.GetGradeScore))
	sg.GET("/getGradeType", authMiddleware, ginx.WrapClaims(h.GetGradeType))
	sg.GET("/getRankByTerm", authMiddleware, ginx.WrapClaimsAndReq(h.GetRankByTerm))
	sg.GET("/loadRank", authMiddleware, ginx.WrapClaims(h.LoadRank))
}

// GetGradeByTerm 查询按学年和学期的成绩
// @Summary 查询按学年和学期的成绩
// @Description 根据学年号和学期号获取用户的成绩,为了方便前端发送请求改成post了
// @Tags grade
// @Accept json
// @Produce json
// @Param data body GetGradeByTermReq  true "获取学年和学期的成绩请求参数"
// @Success 200 {object} web.Response{data=GetGradeByTermResp} "成功返回学年和学期的成绩信息"
// @Failure 500 {object} web.Response "系统异常，获取失败"
// @Router /grade/getGradeByTerm [post]
func (h *GradeHandler) GetGradeByTerm(ctx *gin.Context, req GetGradeByTermReq, uc ijwt.UserClaims) (web.Response, error) {
	if len(req.Kcxzmcs) == 0 {
		return web.Response{
			Msg:  "获取成绩成功!",
			Data: GetGradeByTermResp{},
		}, nil
	}
	grades, err := h.GradeClient.GetGradeByTerm(ctx, &gradev1.GetGradeByTermReq{
		StudentId: uc.StudentId,
		Terms:     convTermsToProto(req.Terms),
		Kcxzmcs:   h.changeKCXZMCS(req.Kcxzmcs),
		Refresh:   req.Refresh,
	})
	if err != nil {
		return web.Response{}, errs.GET_GRADE_BY_TERM_ERROR(err)
	}

	var resp GetGradeByTermResp
	for _, grade := range grades.Grades {
		resp.Grades = append(resp.Grades, Grade{
			Xnm:                 grade.Xnm,
			Xqm:                 grade.Xqm,
			Kcmc:                grade.Kcmc,                         // 课程名
			Xf:                  grade.Xf,                           // 学分
			Jd:                  grade.Jd,                           // 绩点
			Cj:                  grade.Cj,                           // 总成绩
			Kcxzmc:              grade.Kcxzmc,                       // 课程性质名称 比如专业主干课程/通识必修课
			Kclbmc:              grade.Kclbmc,                       // 课程类别名称，比如专业课/公共课
			Kcbj:                grade.Kcbj,                         // 课程标记，比如主修/辅修
			RegularGradePercent: "平时成绩" + grade.RegularGradePercent, // 平时分占比
			RegularGrade:        grade.RegularGrade,                 // 平时分分数
			FinalGradePercent:   "期末成绩" + grade.FinalGradePercent,   // 期末占比
			FinalGrade:          grade.FinalGrade,                   // 期末分数
		})
	}

	// 这里做了一个异步的增加用户的feedCount
	go func() {
		ct := context.Background()
		_, err := h.CounterClient.AddCounter(ct, &counterv1.AddCounterReq{StudentId: uc.StudentId})
		if err != nil {
			h.l.Error("增加用户feedCount失败:", logger.Error(err))
		}
	}()

	return web.Response{
		Msg:  fmt.Sprintf("获取成绩成功!"),
		Data: resp,
	}, nil
}

// 为了解决教务系统的课程性质名称问题
func (h *GradeHandler) changeKCXZMCS(kcxzmcs []string) []string {
	if len(kcxzmcs) == 0 {
		return kcxzmcs
	}

	// 旧 -> 新 映射表
	mapping := map[string]string{
		"专业主干课程":   "专业主干课程",
		"通识必修课":    "通识必修课",
		"通识选修课":    "通识选修课",
		"个性发展课程":   "个性发展课",
		"通识核心课":    "通识核心课",
		"专业选修课":    "专业选修课",
		"教师教育必修":   "教师教育必修课",
		"教师教育选修":   "教师教育选修课",
		"公共必修课":    "公共必修课",
		"必修":       "必修",
		"选修":       "选修",
		"大学英语分级教学": "大学英语分级教学",
	}

	// 额外归一化（老系统可能出现的模糊值）
	alias := map[string]string{
		"专业必修课": "专业必修课",
		"专业必修":  "专业必修课",
		"选修课":   "选修课",
		"选修课程":  "选修课",
		"专业课":   "专业课",
	}

	result := make([]string, 0, len(kcxzmcs))
	seen := make(map[string]struct{})

	for _, old := range kcxzmcs {
		old = strings.TrimSpace(old)

		var newName string

		if v, ok := mapping[old]; ok {
			newName = v
		} else if v, ok := alias[old]; ok {
			newName = v
		} else {
			// 未识别的直接透传，避免影响查询
			newName = old
		}

		if _, ok := seen[newName]; !ok {
			seen[newName] = struct{}{}
			result = append(result, newName)
		}
	}

	return result
}

// GetGradeScore 查询学分
// @Summary 查询学分
// @Description 查询学分
// @Tags grade
// @Accept json
// @Produce json
// @Success 200 {object} web.Response{data=GetGradeScoreResp} "成功返回学分"
// @Failure 500 {object} web.Response "系统异常，获取失败"
// @Router /grade/getGradeScore [get]
func (h *GradeHandler) GetGradeScore(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	// 调用 GradeClient 获取成绩数据
	score, err := h.GradeClient.GetGradeScore(ctx, &gradev1.GetGradeScoreReq{
		StudentId: uc.StudentId,
	})
	if err != nil {
		return web.Response{}, errs.GET_GRADE_SCORE_ERROR(err)
	}

	// 转换为目标结构体
	var resp GetGradeScoreResp
	for _, grade := range score.TypeOfGradeScore {
		typeOfGradeScore := TypeOfGradeScore{
			Kcxzmc:         grade.Kcxzmc,
			GradeScoreList: make([]*GradeScore, len(grade.GradeScoreList)),
		}

		for i := range grade.GradeScoreList {
			typeOfGradeScore.GradeScoreList[i] = &GradeScore{
				// 根据 GradeScore 的字段进行赋值
				Kcmc: grade.GradeScoreList[i].Kcmc,
				Xf:   grade.GradeScoreList[i].Xf,
			}
		}

		resp.TypeOfGradeScores = append(resp.TypeOfGradeScores, typeOfGradeScore)
	}

	return web.Response{
		Data: resp,
	}, nil
}

func convTermsToProto(terms []string) []*gradev1.Terms {
	termMap := make(map[int64]map[int64]struct{})

	for _, termStr := range terms {
		parts := strings.Split(termStr, "-")
		if len(parts) != 2 {
			continue // 非法格式，跳过
		}

		xnm, err1 := strconv.ParseInt(parts[0], 10, 64)
		xqm, err2 := strconv.ParseInt(parts[1], 10, 64)
		if err1 != nil || err2 != nil {
			continue // 非法数字，跳过
		}

		if _, ok := termMap[xnm]; !ok {
			termMap[xnm] = make(map[int64]struct{})
		}
		termMap[xnm][xqm] = struct{}{}
	}

	// 构造 []*gradev1.Terms
	var result []*gradev1.Terms
	for xnm, xqmsSet := range termMap {
		var xqms []int64
		for xqm := range xqmsSet {
			xqms = append(xqms, xqm)
		}
		result = append(result, &gradev1.Terms{
			Xnm:  xnm,
			Xqms: xqms,
		})
	}

	return result
}

// GetGradeType 获取课程类别
// @Summary 获取课程类别
// @Description 获取课程类别
// @Tags grade
// @Accept json
// @Produce json
// @Success 200 {object} web.Response{data=GetGradeTypeResp} "成功返回课程列表"
// @Failure 500 {object} web.Response "系统异常，获取失败"
// @Router /grade/getGradeType [get]
func (h *GradeHandler) GetGradeType(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	list, err := h.GradeClient.GetGradeType(ctx, &gradev1.GetGradeTypeReq{StudentId: uc.StudentId})
	if err != nil {
		return web.Response{}, errs.GET_GRADE_TYPE_ERROR(err)
	}

	var resp GetGradeTypeResp
	resp.Kcxzmc = list.GradeTypes

	return web.Response{
		Msg:  "获取课程类别成功！",
		Data: resp,
	}, nil
}
