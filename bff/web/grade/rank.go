package grade

import (
	"log"

	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
)

const (
	DefaultXnmBegin = 2005 //查询范围开始为2005年
	DefaultXnmEnd   = 2100 //查询范围直到2100年
	DefaultXqmBegin = 1
	DefaultXqmEnd   = 3
)

// GetRankByTerm 查询学分绩排名
// @Summary 查询学分绩排名
// @Description 根据学年号和学期号获取用户的学分绩排名以及分数和统计的科目，全为0则查总排名
// @Tags grade
// @Accept json
// @Produce json
// @Param data body GetRankByTermReq  true "获取学年和学期的学分绩排名请求参数"
// @Success 200 {object} web.Response{data=GetRankByTermResp} "成功返回学年和学期的排名信息"
// @Failure 500 {object} web.Response "系统异常，获取失败"
// @Router /grade/getRankByTerm [get]
func (h *GradeHandler) GetRankByTerm(ctx *gin.Context, req GetRankByTermReq, uc ijwt.UserClaims) (web.Response, error) {
	// 为0则查全学期总排名
	if req.XnmBegin == 0 {
		req.XqmBegin = DefaultXqmBegin
		req.XqmEnd = DefaultXqmEnd
		req.XnmEnd = DefaultXnmEnd
		req.XnmBegin = DefaultXnmBegin
	}

	rank, err := h.GradeClient.GetRankByTerm(ctx, &v1.GetRankByTermReq{
		StudentId: uc.StudentId,
		XnmBegin:  req.XnmBegin,
		XnmEnd:    req.XnmEnd,
		XqmBegin:  req.XqmBegin,
		XqmEnd:    req.XqmEnd,
		Refresh:   req.Refresh,
	})

	if err != nil {
		log.Println(err)
		return web.Response{}, errs.GET_RANK_BY_TERM_ERROR(err)
	}

	resp := &v1.GetRankByTermResp{
		Score:   rank.Score,
		Rank:    rank.Rank,
		Include: rank.Include,
	}
	return web.Response{
		Msg:  "获取排名成功",
		Data: resp,
	}, nil
}

// LoadRank 预加载学分绩排名
// @Summary 预加载总排名
// @Description 当用户点开app时前端发现从未预加载过，调用该接口预加载总排名，每个用户只需调用一次即可
// @Tags grade
// @Produce json
// @Success 200 {object} web.Response "返回信息"
// @Router /grade/loadRank [get]
func (h *GradeHandler) LoadRank(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	// 无需处理错误
	h.GradeClient.LoadRank(ctx, &v1.LoadRankReq{
		StudentId: uc.StudentId,
	})

	return web.Response{
		Msg: "预加载排名操作成功开启",
	}, nil
}
