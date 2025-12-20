package ijwt

import (
	"github.com/gin-gonic/gin"
)

//go:generate mockgen -source=./types.go -package=ijwtmocks -destination=./mocks/ijwt.mock.go Handler
type Handler interface {
	ClearToken(ctx *gin.Context) error
	ExtractToken(ctx *gin.Context) string
	SetLoginToken(ctx *gin.Context, studentId string, password string) error
	SetJWTToken(ctx *gin.Context, cp ClaimParams) error
	CheckSession(ctx *gin.Context, ssid string) (bool, error)
	JWTKey() []byte
	RCJWTKey() []byte
	EncKey() []byte
	DecryptPasswordFromClaims(uc *UserClaims) (string, error)
}

type ClaimParams struct {
	StudentId string
	Password  string
	Ssid      string
	UserAgent string
}
