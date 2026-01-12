package user

import (
	"errors"
	"fmt"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// user板块的控制路由
type UserHandler struct {
	ijwt.Handler
	userClient userv1.UserServiceClient
	preLoader  PreLoader
}

func NewUserHandler(
	hdl ijwt.Handler,
	userSvc userv1.UserServiceClient,
	preLoader PreLoader,

) *UserHandler {

	return &UserHandler{
		Handler:    hdl,
		userClient: userSvc,
		preLoader:  preLoader,
	}
}

// 注册user的路由
func (h *UserHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	ug := s.Group("/users")
	ug.POST("/login_ccnu", ginx.WrapReq(h.LoginByCCNU))
	ug.GET("/logout", authMiddleware, ginx.Wrap(h.Logout))
	ug.GET("/refresh_token", ginx.Wrap(h.RefreshToken))
	ug.POST("/deactivate", authMiddleware, ginx.WrapClaimsAndReq(h.DeleteAccount))
}

// LoginByCCNU
// @Summary ccnu登录
// @Description 通过学号和密码进行登录认证
// @Tags user
// @Accept json
// @Produce json
// @Param request body LoginByCCNUReq true "登录请求体"
// @Success 200 {object} web.Response "Success"
// @Router /users/login_ccnu [post]
func (h *UserHandler) LoginByCCNU(ctx *gin.Context, req LoginByCCNUReq) (web.Response, error) {
	// 记录学号存入 span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("student_id", req.StudentId))

	// 检测是否学生证账号密码正确,如果通行证失败的话会去查本地,如果本地也失败就会丢出系统异常错误,否则是账号密码不正确
	resp, err := h.userClient.CheckUser(ctx, &userv1.CheckUserReq{
		StudentId: req.StudentId,
		Password:  req.Password,
	})
	switch {
	case err == nil:
	// 直接向下执行
	case userv1.IsIncorrectPasswordError(err):
		return web.Response{}, errs.USER_SID_Or_PASSPORD_ERROR(err)
	default:
		return web.Response{}, errs.LOGIN_BY_CCNU_ERROR(err)
	}

	// 兜底的判断
	if !resp.Success {
		return web.Response{}, errs.USER_SID_Or_PASSPORD_ERROR(err)
	}

	// 更新用户账号密码
	_, err = h.userClient.SaveUser(ctx, &userv1.SaveUserReq{StudentId: req.StudentId, Password: req.Password})
	if err != nil {
		return web.Response{}, errs.LOGIN_BY_CCNU_ERROR(err)
	}

	// 预加载基础配置和数据
	h.preLoader.PreLoad(ctx, req.StudentId)

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
// @Tags user
// @Accept json
// @Produce json
// @Success 200 {object} web.Response "Success"
// @Router /users/logout [get]
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
// @Tags user
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
	// 这里设置到相应头里了(非常神秘的模式),这里的jwt参数居然直接被耦合到服务里面去了
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

// DeleteAccount
// @Summary 注销账户
// @Description 用户输入密码验明身份后注销
// @Tags user
// @Accept json
// @Produce json
// @Security Bearer Auth
// @Param request body DeleteAccountReq true "注销账户请求体"
// @Success 200 {object} web.Response "Success"
// @Router /users/deactivate [post]
func (h *UserHandler) DeleteAccount(ctx *gin.Context, req DeleteAccountReq, cla ijwt.UserClaims) (web.Response, error) {
	// todo:这里目前只是伪逻辑，具体的身份验证、软删除、恢复码、恢复码等需要后续实现
	// todo: 通过数据库比较输入和用户真正密码,目前仅是判断是否为空
	if cla.Password == "" {
		fmt.Println(req.Password, "---", cla.Password)
		return web.Response{}, errs.USER_SID_Or_PASSPORD_ERROR(errors.New("password do not match"))
	}

	err := h.ClearToken(ctx)
	if err != nil {
		return web.Response{}, errs.JWT_SYSTEM_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}
