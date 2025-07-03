package banner

import (
	"context"
	"fmt"
	bannerv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/banner/v1"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

// BannerHandler 处理与 banner 相关的 API 请求
type BannerHandler struct {
	bannerClient   bannerv1.BannerServiceClient
	userClient     userv1.UserServiceClient
	Administrators map[string]struct{}
}

// NewBannerHandler 创建一个新的 BannerHandler 实例
func NewBannerHandler(bannerClient bannerv1.BannerServiceClient,
	userClient userv1.UserServiceClient,
	administrators map[string]struct{}) *BannerHandler {
	return &BannerHandler{bannerClient: bannerClient, userClient: userClient, Administrators: administrators}
}

// RegisterRoutes 注册与 banner 相关的路由
func (h *BannerHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/banner")
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
func (h *BannerHandler) GetBanners(ctx *gin.Context, uc ijwt.UserClaims) (web.Response, error) {

	go func() {
		//此处做一个cookie预热
		//为什么在这里做呢?
		//因为用户打开匣子必然会发送这个请求,如果短时间(5分钟)内要获取课表或者是成绩会体验感好很多
		_, _ = h.userClient.GetCookie(context.Background(), &userv1.GetCookieRequest{StudentId: uc.StudentId})
	}()

	banners, err := h.bannerClient.GetBanners(ctx, &bannerv1.GetBannersRequest{})
	if err != nil {
		return web.Response{}, errs.GET_BANNER_ERROR(err)
	}

	//类型转换
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
func (h *BannerHandler) SaveBanner(ctx *gin.Context, req SaveBannerRequest, uc ijwt.UserClaims) (web.Response, error) {

	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.bannerClient.SaveBanner(ctx, &bannerv1.SaveBannerRequest{
		Id:          req.Id,
		PictureLink: req.PictureLink,
		WebLink:     req.WebLink,
	})
	if err != nil {
		return web.Response{}, errs.Save_BANNER_ERROR(err)
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
func (h *BannerHandler) DelBanner(ctx *gin.Context, req DelBannerRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.bannerClient.DelBanner(ctx, &bannerv1.DelBannerRequest{Id: req.Id})
	if err != nil {
		return web.Response{}, errs.Del_BANNER_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

func (h *BannerHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
