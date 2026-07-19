package content

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

func (h *ContentHandler) RegisterBannerRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/banner")
	sg.GET("/getBanners", authMiddleware, ginx.WrapClaims(h.GetBanners))
	sg.POST("/saveBanner", authMiddleware, ginx.WrapClaimsAndReq(h.SaveBanner))
	sg.POST("/delBanner", authMiddleware, ginx.WrapClaimsAndReq(h.DelBanner))
}

// GetBanners 获取 banner 列表
// @Summary 获取 banner 列表
// @Description 获取 banner 列表
// @Tags banner
// @Success 200 {object} web.Response{data=GetBannersResponse} "成功"
// @Router /banner/getBanners [get]
func (h *ContentHandler) GetBanners(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {
	go func() {
		//这里ctx不能使用ctx.Request.Context(),这个ctx会在http请求结束时直接取消，导致协程中的这两步并没有生效
		reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// 此处做一个cookie预热和一个成绩预加载
		// 为什么在这里做呢? 好问题
		// 因为用户打开匣子必然会发送这个请求,如果短时间(5分钟)内要获取课表或者是成绩会体验感好很多
		_, _ = h.userClient.GetCookie(reqCtx, &userv1.GetCookieRequest{StudentId: uc.StudentId})
		_, _ = h.counterClient.AddCounter(reqCtx, &counterv1.AddCounterReq{StudentId: uc.StudentId})
	}()

	banners, err := h.contentClient.GetBanners(ctx, &contentv1.GetBannersRequest{})
	if err != nil {
		return web.Response{}, errs.GET_BANNER_ERROR(err)
	}

	// 类型转换
	var resp GetBannersResponse
	err = copier.Copy(&resp.Banners, &banners.Banners)
	if err != nil {
		return web.Response{}, errs.GET_BANNER_ERROR(err)
	}
	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// SaveBanner 保存 banner 内容
// @Summary 保存 banner 内容
// @Description 保存 banner 内容,如果不添加id字段表示添加一个新的banner
// @Tags banner
// @Accept json
// @Produce json
// @Param request body SaveBannerRequest true "保存 banner 内容请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /banner/saveBanner [post]
func (h *ContentHandler) SaveBanner(ctx *gin.Context, req SaveBannerRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.contentClient.SaveBanner(ctx, &contentv1.SaveBannerRequest{
		Id:          req.Id,
		PictureLink: req.PictureLink,
		WebLink:     req.WebLink,
	})
	if err != nil {
		return web.Response{}, errs.SAVE_BANNER_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

// DelBanner 删除 banner 内容
// @Summary 删除 banner 内容
// @Description 删除 banner 内容
// @Tags banner
// @Accept json
// @Produce json
// @Param request body DelBannerRequest true "删除 banner 内容请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /banner/delBanner [post]
func (h *ContentHandler) DelBanner(ctx *gin.Context, req DelBannerRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.contentClient.DelBanner(ctx, &contentv1.DelBannerRequest{Id: req.Id})
	if err != nil {
		return web.Response{}, errs.DEL_BANNER_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}
