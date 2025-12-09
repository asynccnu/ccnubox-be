package ijwt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-pkg/ginx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisJWTHandler 实现了处理 JWT 的接口，并使用 Redis 进行支持
type RedisJWTHandler struct {
	cmd           redis.Cmdable     // Redis 命令接口，用于与 Redis 进行交互
	signingMethod jwt.SigningMethod // JWT 签名方法
	rcExpiration  time.Duration     // 刷新令牌的过期时间，防止缓存过大
	jwtKey        []byte            // 用于签署 JWT 的密钥
	rcJWTKey      []byte            // 用于签署刷新令牌的密钥
	encKey        []byte            // 用于加密敏感信息（密码）的密钥
}

// JWTKey 返回用于签署 JWT 的密钥
func (r *RedisJWTHandler) JWTKey() []byte {
	return r.jwtKey
}

// RCJWTKey 返回用于签署刷新令牌的密钥
func (r *RedisJWTHandler) RCJWTKey() []byte {
	return r.rcJWTKey
}

func (r *RedisJWTHandler) EncKey() []byte {
	return r.encKey
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
	enPassword, err := r.encryptString(password)
	if err != nil {
		return err
	}
	cp := ClaimParams{
		StudentId: studentId,
		Password:  enPassword,
		Ssid:      uuid.New().String(),
		UserAgent: ctx.GetHeader("User-Agent"),
	}
	err = r.setRefreshToken(ctx, cp)
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
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
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
func NewRedisJWTHandler(cmd redis.Cmdable, jwtKey string, rcJWTKey string, encKey string) Handler {
	return &RedisJWTHandler{
		cmd:           cmd,                     //redis实体
		signingMethod: jwt.SigningMethodHS256,  //签名的加密方式
		rcExpiration:  time.Hour * 24 * 30 * 6, //设置为六个月之后过期
		jwtKey:        []byte(jwtKey),
		rcJWTKey:      []byte(rcJWTKey),
		encKey:        []byte(encKey),
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

// 辅助：用 sha256 派生 32 字节 key
func deriveKey(key []byte) []byte {
	h := sha256.Sum256(key)
	return h[:]
}

// encryptString 使用 AES-GCM 将明文加密并返回 base64( nonce | ciphertext )
func (r *RedisJWTHandler) encryptString(plain string) (string, error) {
	key := deriveKey(r.encKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, []byte(plain), nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// decryptString 解密 base64( nonce | ciphertext ) 并返回明文
func (r *RedisJWTHandler) decryptString(b64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	key := deriveKey(r.encKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := data[:ns], data[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// DecryptPasswordFromClaims 对外：根据 UserClaims 解密出明文 password（供后续业务使用，先留一个钩子）
func (r *RedisJWTHandler) DecryptPasswordFromClaims(uc *UserClaims) (string, error) {
	if uc == nil || uc.Password == "" {
		return "", nil
	}
	return r.decryptString(uc.Password)
}
