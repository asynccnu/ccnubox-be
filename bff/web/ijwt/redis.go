package ijwt

import (
	"fmt"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

// RedisJWTHandler 实现了处理 JWT 的接口，并使用 Redis 进行支持
type RedisJWTHandler struct {
	cmd           redis.Cmdable     // Redis 命令接口，用于与 Redis 进行交互
	signingMethod jwt.SigningMethod // JWT 签名方法
	rcExpiration  time.Duration     // 刷新令牌的过期时间，防止缓存过大
	jwtKey        []byte            // 用于签署 JWT 的密钥
	rcJWTKey      []byte            // 用于签署刷新令牌的密钥
}

// JWTKey 返回用于签署 JWT 的密钥
func (r *RedisJWTHandler) JWTKey() []byte {
	return r.jwtKey
}

// RCJWTKey 返回用于签署刷新令牌的密钥
func (r *RedisJWTHandler) RCJWTKey() []byte {
	return r.rcJWTKey
}

// ClearToken 清除客户端的 JWT 和刷新令牌，并在 Redis 中记录已过期的会话
func (r *RedisJWTHandler) ClearToken(ctx *gin.Context) error {
	// 要求客户端设置为空
	ctx.Header("x-jwt-token", "")
	ctx.Header("x-refresh-token", "")
	// 在 Redis 中记录已过期的会话 TODO 这里需要解耦合,但是写的太抽象了一时半会儿看不明白,先这么做
	uc, err := ginx.GetClaims[UserClaims](ctx)
	if err != nil {
		return err
	}

	return r.cmd.Set(ctx, fmt.Sprintf("ccnubox:users:ssid:%s", uc.Ssid), "", r.rcExpiration).Err()
}

// ExtractToken 从请求中提取并返回 JWT
func (r *RedisJWTHandler) ExtractToken(ctx *gin.Context) string {
	authCode := ctx.GetHeader("Authorization")
	if authCode == "" {
		return ""
	}
	segs := strings.Split(authCode, " ")
	if len(segs) != 2 {
		return ""
	}
	return segs[1]
}

// SetLoginToken 设置用户的刷新令牌和 JWT
func (r *RedisJWTHandler) SetLoginToken(ctx *gin.Context, studentId string, password string) error {
	cp := ClaimParams{
		StudentId: studentId,
		Password:  password,
		Ssid:      uuid.New().String(),
		UserAgent: ctx.GetHeader("User-Agent"),
	}
	err := r.setRefreshToken(ctx, cp)
	if err != nil {
		return err
	}
	return r.SetJWTToken(ctx, cp)
}

// setRefreshToken 生成并设置用户的刷新令牌
func (r *RedisJWTHandler) setRefreshToken(ctx *gin.Context, cp ClaimParams) error {
	rc := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(r.rcExpiration)),
		},
		StudentId: cp.StudentId,
		Password:  cp.Password,
		Ssid:      cp.Ssid,
		UserAgent: cp.UserAgent,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rc)
	tokenStr, err := token.SignedString(r.RCJWTKey())
	if err != nil {
		return err
	}
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}

// SetJWTToken 生成并设置用户的 JWT
func (r *RedisJWTHandler) SetJWTToken(ctx *gin.Context, cp ClaimParams) error {
	uc := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)), //一天过期一次
		},
		StudentId: cp.StudentId,
		Password:  cp.Password,
		Ssid:      cp.Ssid,
		UserAgent: cp.UserAgent,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, uc)
	tokenStr, err := token.SignedString(r.JWTKey())
	if err != nil {
		return err
	}
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

// CheckSession 检查给定 ssid 的会话是否有效
func (r *RedisJWTHandler) CheckSession(ctx *gin.Context, ssid string) (bool, error) {
	val, err := r.cmd.Exists(ctx, fmt.Sprintf("ccnubox:users:ssid:%s", ssid)).Result()
	return val > 0, err
}

// NewRedisJWTHandler 创建并返回一个新的 RedisJWTHandler 实例
func NewRedisJWTHandler(cmd redis.Cmdable, jwtKey string, rcJWTKey string) Handler {
	return &RedisJWTHandler{
		cmd:           cmd,                    //redis实体
		signingMethod: jwt.SigningMethodHS256, //签名的加密方式
		rcExpiration:  time.Hour * 24 * 7,     //设置为一周之后过期
		jwtKey:        []byte(jwtKey),
		rcJWTKey:      []byte(rcJWTKey),
	}
}

// UserClaims 定义了 JWT 中用户相关的声明
type UserClaims struct {
	jwt.RegisteredClaims
	StudentId string // 学生 ID
	Password  string // 密码（仅用于演示，实际应用中不会存储密码）
	Ssid      string // 会话 ID
	UserAgent string // 用户代理信息
}

// RefreshClaims 定义了刷新令牌中的声明
type RefreshClaims struct {
	jwt.RegisteredClaims
	StudentId string // 学生 ID
	Password  string // 密码
	Ssid      string // 会话 ID
	UserAgent string // 用户代理信息
}
