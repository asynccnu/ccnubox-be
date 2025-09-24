package grade

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	counterv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/counter/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/logger"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

type GradeHandler struct {
	GradeClient    gradev1.GradeServiceClient //注入的是grpc服务
	CounterClient  counterv1.CounterServiceClient
	Administrators map[string]struct{} //这里注入的是管理员权限验证配置
	l              logger.Logger
}

func NewGradeHandler(
	GradeClient gradev1.GradeServiceClient, //注入的是grpc服务
	CounterClient counterv1.CounterServiceClient,
	l logger.Logger,
	administrators map[string]struct{}) *GradeHandler {
	return &GradeHandler{
		GradeClient:    GradeClient,
		CounterClient:  CounterClient,
		Administrators: administrators,
		l:              l,
	}
}

func (h *GradeHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/grade")
	//这里有三类路由,分别是ginx.WrapClaimsAndReq()有参数且要验证
	sg.POST("/getGradeByTerm", authMiddleware, ginx.WrapClaimsAndReq(h.GetGradeByTerm))
	sg.GET("/getGradeScore", authMiddleware, ginx.WrapClaims(h.GetGradeScore))
	sg.POST("/getGraduateGrade", authMiddleware, ginx.WrapClaimsAndReq(h.UpdateGraduateGrades))
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
	grades, err := h.GradeClient.GetGradeByTerm(ctx, &gradev1.GetGradeByTermReq{
		StudentId: uc.StudentId,
		Terms:     convTermsToProto(req.Terms),
		Kcxzmcs:   req.Kcxzmcs,
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
			Kcmc:                grade.Kcmc,                // 课程名
			Xf:                  grade.Xf,                  // 学分
			Jd:                  grade.Jd,                  // 绩点
			Cj:                  grade.Cj,                  // 总成绩
			Kcxzmc:              grade.Kcxzmc,              // 课程性质名称 比如专业主干课程/通识必修课
			Kclbmc:              grade.Kclbmc,              // 课程类别名称，比如专业课/公共课
			Kcbj:                grade.Kcbj,                // 课程标记，比如主修/辅修
			RegularGradePercent: grade.RegularGradePercent, // 平时分占比
			RegularGrade:        grade.RegularGrade,        // 平时分分数
			FinalGradePercent:   grade.FinalGradePercent,   // 期末占比
			FinalGrade:          grade.FinalGrade,          // 期末分数
		})
	}

	//这里做了一个异步的增加用户的feedCount
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

// UpdateGraduateGrades 查询研究生成绩
// @Summary 查询研究生成绩
// @Description 根据学年号和学期号获取用户的成绩
// @Tags grade
// @Accept json
// @Produce json
// @Param data body UpdateGraduateGradesReq  true "获取学年和学期的成绩请求参数"
// @Success 200 {object} web.Response{data=UpdateGraduateGradesResp} "成功返回学年和学期的成绩信息"
// @Failure 500 {object} web.Response "系统异常，获取失败"
// @Router /grade/getGraduateGrade [post]
func (h *GradeHandler) UpdateGraduateGrades(ctx *gin.Context, req UpdateGraduateGradesReq, uc ijwt.UserClaims) (web.Response, error) {
	grpcResp, err := h.GradeClient.GetGraduateGrade(ctx, &gradev1.GetGraduateUpdateReq{
		StudentId: uc.StudentId,
		Xnm:       req.Xnm,
		Xqm:       req.Xqm,
		Cjzt:      req.Cjzt,
	})
	if err != nil {
		return web.Response{}, errs.GET_GRADE_SCORE_ERROR(err)
	}

	var resp UpdateGraduateGradesResp
	for _, g := range grpcResp.Grades {
		resp.Grades = append(resp.Grades, GraduateGrade{
			JxbId:           g.JxbId,
			Status:          g.Status,
			Year:            g.Year,
			Term:            g.Term,
			Name:            g.Name,
			StudentCategory: g.StudentCategory,
			College:         g.College,
			Major:           g.Major,
			Grade:           g.Grade,
			ClassCode:       g.ClassCode,
			ClassName:       g.ClassName,
			ClassNature:     g.ClassNature,
			Credit:          g.Credit,
			Point:           g.Point,
			GradePoints:     g.GradePoints,
			IsAvailable:     g.IsAvailable,
			IsDegree:        g.IsDegree,
			SetCollege:      g.SetCollege,
			ClassMark:       g.ClassMark,
			ClassCategory:   g.ClassCategory,
			ClassID:         g.ClassID,
			Teacher:         g.Teacher,
		})
	}
	return web.Response{Data: resp}, nil
}
