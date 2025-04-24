package user

import (
	ccnuv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/ccnu/v1"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// user板块的控制路由
type UserHandler struct {
	ijwt.Handler
	userSvc userv1.UserServiceClient
	ccnuSvc ccnuv1.CCNUServiceClient
}

func NewUserHandler(hdl ijwt.Handler, userSvc userv1.UserServiceClient, ccnuSvc ccnuv1.CCNUServiceClient) *UserHandler {
	return &UserHandler{
		Handler: hdl,
		userSvc: userSvc,
		ccnuSvc: ccnuSvc,
	}
}

// 注册user的路由
func (h *UserHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	ug := s.Group("/users")
	ug.POST("/login_ccnu", ginx.WrapReq(h.LoginByCCNU))
	ug.POST("/logout", authMiddleware, ginx.Wrap(h.Logout))
	ug.GET("/refresh_token", ginx.Wrap(h.RefreshToken))
}

// LoginByCCNU
// @Summary ccnu登录
// @Description 通过学号和密码进行登录认证
// @Tags 用户
// @Accept json
// @Produce json
// @Param request body LoginByCCNUReq true "登录请求体"
// @Success 200 {object} web.Response "Success"
// @Router /users/login_ccnu [post]
func (h *UserHandler) LoginByCCNU(ctx *gin.Context, req LoginByCCNUReq) (web.Response, error) {
	//请求部分,内层调用的be-api上面的东西,实现grpc通信可以直接点进去看
	_, err := h.ccnuSvc.Login(ctx, &ccnuv1.LoginRequest{
		StudentId: req.StudentId,
		Password:  req.Password,
	})
	switch {
	case err == nil:
	// 直接向下执行
	case ccnuv1.IsInvalidSidOrPwd(err):
		return web.Response{}, errs.USER_SID_Or_PASSPORD_ERROR(err)
	default:
		return web.Response{}, errs.LOGIN_BY_CCNU_ERROR(err)
	}
	// FindOrCreate
	_, err = h.userSvc.SaveUser(ctx, &userv1.SaveUserReq{StudentId: req.StudentId, Password: req.Password})
	if err != nil {
		return web.Response{}, errs.LOGIN_BY_CCNU_ERROR(err)
	}

	err = h.SetLoginToken(ctx, req.StudentId, req.Password)
	if err != nil {
		return web.Response{}, errs.JWT_SYSTEM_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// Logout
// @Summary 登出(销毁token)
// @Description 通过短token登出
// @Tags 用户
// @Accept json
// @Produce json
// @Success 200 {object} web.Response "Success"
// @Router /users/logout [post]
func (h *UserHandler) Logout(ctx *gin.Context) (web.Response, error) {
	err := h.ClearToken(ctx)
	if err != nil {
		return web.Response{}, errs.JWT_SYSTEM_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

// RefreshToken
// @Summary 刷新短token
// @Description 通过长token刷新短token
// @Tags 用户
// @Accept json
// @Produce json
// @Security Bearer Auth
// @Success 200 {object} web.Response "Success"
// @Router /users/refresh_token [get]
func (h *UserHandler) RefreshToken(ctx *gin.Context) (web.Response, error) {
	tokenStr := h.ExtractToken(ctx)
	rc := &ijwt.RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, rc, func(*jwt.Token) (interface{}, error) {
		// 可以根据具体情况给出不同的key
		return h.RCJWTKey(), nil
	})
	if err != nil {
		return web.Response{}, errs.AUTH_PASSED_ERROR(err)
	}
	if token == nil || !token.Valid {
		return web.Response{}, errs.UNAUTHORIED_ERROR(err)
	}
	ok, err := h.CheckSession(ctx, rc.Ssid)
	if err != nil || ok {
		return web.Response{}, errs.JWT_SYSTEM_ERROR(err)
	}
	//这里设置到相应头里了(非常神秘的模式),这里的jwt参数居然直接被耦合到服务里面去了
	err = h.SetJWTToken(ctx, ijwt.ClaimParams{
		StudentId: rc.StudentId,
		Password:  rc.Password,
		Ssid:      rc.Ssid,
		UserAgent: rc.UserAgent,
	})
	if err != nil {
		return web.Response{}, errs.JWT_SYSTEM_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}
