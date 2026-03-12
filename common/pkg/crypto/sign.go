package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildSignWithSecret 构造签名，用于座位预约请求头
func BuildSignWithSecret(method string, secret string) (string, string, int64) {
	id := uuid.New().String()
	ts := time.Now().UnixMilli()

	str := fmt.Sprintf(
		"seat::%s::%d::%s",
		id,
		ts,
		strings.ToUpper(method),
	)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(str))

	sign := hex.EncodeToString(h.Sum(nil))

	return id, sign, ts
}
