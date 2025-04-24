package feed

import (
	"fmt"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"time"
)

type FeedHandler struct {
	feedClient     feedv1.FeedServiceClient
	Administrators map[string]struct{}
}

func NewFeedHandler(feedClient feedv1.FeedServiceClient,
	administrators map[string]struct{}) *FeedHandler {
	return &FeedHandler{feedClient: feedClient, Administrators: administrators}
}

func (h *FeedHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/feed")
	sg.GET("/getFeedEvents", authMiddleware, ginx.WrapClaims(h.GetFeedEvents))
	sg.POST("/clearFeedEvent", authMiddleware, ginx.WrapClaimsAndReq(h.ClearFeedEvent))
	sg.POST("/changeFeedAllowList", authMiddleware, ginx.WrapClaimsAndReq(h.ChangeFeedAllowList))
	sg.GET("/getFeedAllowList", authMiddleware, ginx.WrapClaims(h.GetFeedAllowList))
	sg.POST("/readFeedEvent", authMiddleware, ginx.WrapReq(h.ReadFeedEvent))
	sg.POST("/saveFeedToken", authMiddleware, ginx.WrapClaimsAndReq(h.SaveFeedToken))
	sg.POST("/removeFeedToken", authMiddleware, ginx.WrapClaimsAndReq(h.RemoveFeedToken))
	sg.POST("/publicMuxiOfficialMSG", authMiddleware, ginx.WrapClaimsAndReq(h.PublicMuxiOfficialMSG))
	sg.POST("/stopMuxiOfficialMSG", authMiddleware, ginx.WrapClaimsAndReq(h.StopMuxiOfficialMSG))
	sg.GET("/getToBePublicOfficialMSG", authMiddleware, ginx.WrapClaims(h.GetToBePublicOfficialMSG))
}

// GetFeedEvents
// @Summary 获取feed订阅事件
// @Description 获取已登录用户的所有feed订阅事件（包括已读和未读）
// @Tags feed订阅
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} web.Response{data=GetFeedEventsResp} "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/getFeedEvents [get]
func (h *FeedHandler) GetFeedEvents(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	feeds, err := h.feedClient.GetFeedEvents(ctx, &feedv1.GetFeedEventsReq{StudentId: uc.StudentId})
	if err != nil {
		return web.Response{}, errs.GET_FEED_EVENTS_ERROR(err)
	}

	//类型转换
	var resp GetFeedEventsResp

	err = copier.Copy(&resp, &feeds)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// ClearFeedEvent
// @Summary 清除feed订阅事件
// @Description 清除指定用户的feed订阅事件,都是可选字段
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Param data body ClearFeedEventReq true "feed订阅事件ID"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/clearFeedEvent [post]
func (h *FeedHandler) ClearFeedEvent(ctx *gin.Context, req ClearFeedEventReq, uc ijwt.UserClaims) (web.Response, error) {
	_, err := h.feedClient.ClearFeedEvent(ctx, &feedv1.ClearFeedEventReq{
		StudentId: uc.StudentId, //用户的id
		FeedId:    req.FeedId,
		Status:    req.Status,
	})

	if err != nil {
		return web.Response{}, errs.CLEAR_FEED_EVENT_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

// ReadFeedEvent
// @Summary 标注feed订阅事件为已读
// @Description 标注feed订阅事件为已读
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Param data body ReadFeedEventReq true "feed订阅事件ID"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/readFeedEvent [post]
func (h *FeedHandler) ReadFeedEvent(ctx *gin.Context, req ReadFeedEventReq) (web.Response, error) {
	_, err := h.feedClient.ReadFeedEvent(ctx, &feedv1.ReadFeedEventReq{
		FeedId: req.FeedId,
	})
	if err != nil {
		return web.Response{}, errs.READ_FEED_EVENT_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// ChangeFeedAllowList
// @Summary 修改feed订阅白名单
// @Description 修改已登录用户的feed订阅白名单设置
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Param data body ChangeFeedAllowListReq true "白名单设置"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/changeFeedAllowList [post]
func (h *FeedHandler) ChangeFeedAllowList(ctx *gin.Context, req ChangeFeedAllowListReq, uc ijwt.UserClaims) (web.Response, error) {

	_, err := h.feedClient.ChangeFeedAllowList(ctx, &feedv1.ChangeFeedAllowListReq{
		AllowList: &feedv1.AllowList{
			StudentId: uc.StudentId,
			Grade:     req.Grade,
			Muxi:      req.Muxi,
			Holiday:   req.Holiday,
			Energy:    req.Energy,
		},
	})

	if err != nil {
		return web.Response{}, errs.CHANGE_FEED_ALLOW_LIST_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// GetFeedAllowList
// @Summary 获取feed订阅白名单
// @Description 获取已登录用户的feed订阅白名单设置
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} web.Response{data=GetFeedAllowListResp} "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/getFeedAllowList [get]
func (h *FeedHandler) GetFeedAllowList(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {

	allowlist, err := h.feedClient.GetFeedAllowList(ctx, &feedv1.GetFeedAllowListReq{StudentId: uc.StudentId})
	if err != nil {
		return web.Response{}, errs.GET_FEED_ALLOW_LIST_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
		Data: GetFeedAllowListResp{
			Grade:   allowlist.AllowList.Grade,
			Muxi:    allowlist.AllowList.Muxi,
			Holiday: allowlist.AllowList.Holiday,
			Energy:  allowlist.AllowList.Energy,
		},
	}, nil
}

// SaveFeedToken
// @Summary 保存feed订阅Token
// @Description 保存已登录用户的feed订阅Token
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Param data body SaveFeedTokenReq true "feed订阅Token"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/saveFeedToken [post]
func (h *FeedHandler) SaveFeedToken(ctx *gin.Context, req SaveFeedTokenReq, uc ijwt.UserClaims) (web.Response, error) {
	_, err := h.feedClient.SaveFeedToken(ctx, &feedv1.SaveFeedTokenReq{
		StudentId: uc.StudentId,
		Token:     req.Token,
	})

	if err != nil {
		return web.Response{}, errs.SAVE_FEED_TOKEN_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// RemoveFeedToken
// @Summary 删除feed订阅Token
// @Description 删除已登录用户的feed订阅Token
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Param data body RemoveFeedTokenReq true "feed订阅Token"
// @Success 200 {object} web.Response "成功"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/removeFeedToken [post]
func (h *FeedHandler) RemoveFeedToken(ctx *gin.Context, req RemoveFeedTokenReq, uc ijwt.UserClaims) (web.Response, error) {
	_, err := h.feedClient.RemoveFeedToken(ctx, &feedv1.RemoveFeedTokenReq{StudentId: uc.StudentId, Token: req.Token})
	if err != nil {
		return web.Response{}, errs.REMOVE_FEED_TOKEN_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// PublicMuxiOfficialMSG
// @Summary 发布木犀官方消息
// @Description 发布木犀官方消息，仅限管理员操作
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Param data body PublicMuxiOfficialMSGReq true "木犀官方消息"
// @Success 200 {object} web.Response{data=PublicMuxiOfficialMSGResp} "成功"
// @Failure 403 {object} web.Response "没有访问权限"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/publicMuxiOfficialMSG [post]
func (h *FeedHandler) PublicMuxiOfficialMSG(ctx *gin.Context, req PublicMuxiOfficialMSGReq, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.feedClient.PublicMuxiOfficialMSG(ctx, &feedv1.PublicMuxiOfficialMSGReq{
		MuxiOfficialMSG: &feedv1.MuxiOfficialMSG{
			Title:        req.Title,
			Content:      req.Content,
			ExtendFields: req.ExtendFields,
			PublicTime:   time.Now().Add(time.Duration(req.LaterTime) * time.Second).Unix(),
		},
	})

	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, errs.PUBLIC_MUXI_OFFICIAL_MSG_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

// StopMuxiOfficialMSG
// @Summary 停止木犀官方消息
// @Description 停止木犀官方消息，仅限管理员操作
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Param data body StopMuxiOfficialMSGReq true "停止消息请求"
// @Success 200 {object} web.Response "成功"
// @Failure 403 {object} web.Response "没有访问权限"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/stopMuxiOfficialMSG [post]
func (h *FeedHandler) StopMuxiOfficialMSG(ctx *gin.Context, req StopMuxiOfficialMSGReq, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}
	_, err := h.feedClient.StopMuxiOfficialMSG(ctx, &feedv1.StopMuxiOfficialMSGReq{
		Id: req.Id,
	})

	if err != nil {
		return web.Response{}, errs.STOP_MUXI_OFFICIAL_MSG_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// GetToBePublicOfficialMSG
// @Summary 获取待发布的官方消息
// @Description 获取计划发布的官方消息，仅限管理员操作
// @Tags feed订阅
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} web.Response{data=GetToBePublicMuxiOfficialMSGResp} "成功"
// @Failure 403 {object} web.Response "没有访问权限"
// @Failure 500 {object} web.Response "系统异常"
// @Router /feed/getToBePublicOfficialMSG [get]
func (h *FeedHandler) GetToBePublicOfficialMSG(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	msgs, err := h.feedClient.GetToBePublicOfficialMSG(ctx, &feedv1.GetToBePublicOfficialMSGReq{})
	if err != nil {
		return web.Response{}, errs.GET_TO_BE_PUBLIC_OFFICIAL_MSG_ERROR(err)
	}

	var response GetToBePublicMuxiOfficialMSGResp
	err = copier.Copy(&response.MSGList, &msgs.MsgList)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: response,
	}, nil
}

func (h *FeedHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
