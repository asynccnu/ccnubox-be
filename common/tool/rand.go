package tool

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomMD5() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
