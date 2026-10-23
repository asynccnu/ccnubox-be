package feed

import (
	"encoding/json"
	"errors"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/bff/service/activity"
	feedservice "github.com/asynccnu/ccnubox-be/bff/service/feed"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
)

type SetWhiteListReq struct {
	Strategy string `json:"strategy" binding:"required" enums:"high_activity,script,all" example:"high_activity"`
}
type SetWhiteListResp struct {
	Strategy  string `json:"strategy"`
	UserCount int32  `json:"user_count"`
	// All 表示使用全量在校用户策略，不表示向历史全部账号广播。
	All       bool  `json:"all"`
	ExpiresAt int64 `json:"expires_at"`
}

// SetWhiteList selects recipients using a backend strategy.
// @Summary 设置木犀官方消息发送名单
// @Description 仅限管理员。high_activity 默认筛选最近 7 个完整自然日中至少活跃 5 天的用户（北京时间，不含当天，阈值可由后端配置）；script 执行后端配置的名单脚本；all 筛选系统内全部在校用户，无需等待活跃明细积累。在校用户按学号前四位的入学年份和四年学制判断，每年北京时间 6 月 30 日起排除毕业批次；不接收学号参数。名单有效期为 7 天，后续通过 publicMuxiOfficialMSG 发布。
// @Tags feed
// @Accept json
// @Produce json
// @Param Authorization header string true "管理员访问令牌（Bearer Token）"
// @Param data body SetWhiteListReq true "后端名单筛选策略"
// @Success 200 {object} web.Response{data=SetWhiteListResp} "设置成功，返回实际名单人数"
// @Failure 400 {object} web.Response "参数错误或名单为空"
// @Failure 401 {object} web.Response "未登录或令牌失效"
// @Failure 403 {object} web.Response "没有访问权限"
// @Failure 409 {object} web.Response "活跃明细未就绪或名单不可用，错误信息包含具体原因"
// @Failure 422 {object} web.Response "请求体不合法"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/setWhiteList [post]
func (h *FeedHandler) SetWhiteList(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(errors.New("仅限管理员操作"))
	}
	if ctx.ContentType() != "application/json" {
		return web.Response{}, errs.BAD_ENTITY_ERROR(errors.New("请求体必须为 application/json"))
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 1024)
	d := json.NewDecoder(ctx.Request.Body)
	d.DisallowUnknownFields()
	var req SetWhiteListReq
	if err := d.Decode(&req); err != nil {
		return web.Response{}, errs.BAD_ENTITY_ERROR(err)
	}
	if d.Decode(new(any)) != io.EOF {
		return web.Response{}, errs.BAD_ENTITY_ERROR(errors.New("请求体必须为单个 JSON 对象"))
	}
	result, err := h.whiteList.Set(ctx, req.Strategy)
	if err != nil {
		switch {
		case errors.Is(err, feedservice.ErrInvalidStrategy):
			return web.Response{}, errs.INVALID_PARAM_VALUE_ERROR(err)
		case errors.Is(err, feedservice.ErrEmptyWhiteList):
			return web.Response{}, b_errorx.New(http.StatusBadRequest, errs.WHITE_LIST_EMPTY_ERROR_CODE, "当前策略未筛选到用户")
		case errors.Is(err, activity.ErrNotReady):
			return web.Response{}, b_errorx.New(http.StatusConflict, errs.WHITE_LIST_ACTIVITY_NOT_READY_ERROR_CODE, err.Error())
		case status.Code(err) == codes.FailedPrecondition:
			return web.Response{}, b_errorx.New(http.StatusConflict, errs.WHITE_LIST_NOT_READY_ERROR_CODE, status.Convert(err).Message())
		case status.Code(err) == codes.InvalidArgument:
			return web.Response{}, errs.INVALID_PARAM_VALUE_ERROR(err)
		default:
			return web.Response{}, &b_errorx.CustomError{HttpCode: http.StatusInternalServerError, Code: errs.WHITE_LIST_INTERNAL_ERROR_CODE, Message: "设置发送名单失败", Cause: err}
		}
	}
	return web.Response{Msg: "Success", Data: SetWhiteListResp{Strategy: result.Strategy, UserCount: result.UserCount, All: req.Strategy == "all", ExpiresAt: result.ExpiresAt}}, nil
}
