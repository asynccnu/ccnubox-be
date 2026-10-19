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
	"io"
	"net/http"
)

type SetWhiteListReq struct {
	Strategy string `json:"strategy" binding:"required" enums:"high_activity,script,all" example:"high_activity"`
}
type SetWhiteListResp struct {
	Strategy  string `json:"strategy"`
	UserCount int32  `json:"user_count"`
	All       bool   `json:"all"`
	ExpiresAt int64  `json:"expires_at"`
}

// SetWhiteList selects recipients using a backend strategy.
// @Summary Set official message white list
// @Description Admin only. high_activity uses backend activity rules; script runs the configured backend command; all explicitly selects every user. No student IDs are accepted. The list is valid for 7 days and applies to subsequent publicMuxiOfficialMSG calls.
// @Tags feed
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param data body SetWhiteListReq true "Backend selection strategy"
// @Success 200 {object} web.Response{data=SetWhiteListResp}
// @Failure 400 {object} web.Response
// @Failure 401 {object} web.Response
// @Failure 403 {object} web.Response
// @Failure 409 {object} web.Response
// @Failure 422 {object} web.Response
// @Failure 500 {object} web.Response
// @Router /feed/setWhiteList [post]
func (h *FeedHandler) SetWhiteList(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(errors.New("admin required"))
	}
	if ctx.ContentType() != "application/json" {
		return web.Response{}, errs.BAD_ENTITY_ERROR(errors.New("application/json required"))
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 1024)
	d := json.NewDecoder(ctx.Request.Body)
	d.DisallowUnknownFields()
	var req SetWhiteListReq
	if err := d.Decode(&req); err != nil {
		return web.Response{}, errs.BAD_ENTITY_ERROR(err)
	}
	if d.Decode(new(any)) != io.EOF {
		return web.Response{}, errs.BAD_ENTITY_ERROR(errors.New("one JSON object required"))
	}
	result, err := h.whiteList.Set(ctx, req.Strategy)
	if err != nil {
		switch {
		case errors.Is(err, feedservice.ErrInvalidStrategy):
			return web.Response{}, errs.INVALID_PARAM_VALUE_ERROR(err)
		case errors.Is(err, feedservice.ErrEmptyWhiteList):
			return web.Response{}, b_errorx.New(http.StatusBadRequest, errs.WHITE_LIST_EMPTY_ERROR_CODE, "Strategy selected no users")
		case errors.Is(err, activity.ErrNotReady):
			return web.Response{}, b_errorx.New(http.StatusConflict, errs.WHITE_LIST_ACTIVITY_NOT_READY_ERROR_CODE, err.Error())
		default:
			return web.Response{}, &b_errorx.CustomError{HttpCode: http.StatusInternalServerError, Code: errs.WHITE_LIST_INTERNAL_ERROR_CODE, Message: "Set white list failed", Cause: err}
		}
	}
	return web.Response{Msg: "Success", Data: SetWhiteListResp{Strategy: result.Strategy, UserCount: result.UserCount, All: req.Strategy == "all", ExpiresAt: result.ExpiresAt}}, nil
}
