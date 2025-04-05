package ioc

import (
	"github.com/asynccnu/ccnubox-be/be-user/pkg/crypto"
	"log"
	"os"
)

// NewCrypto 创建一个新的 Crypto 实例，key 必须是 16, 24, 或 32 字节长度（对应 AES-128, AES-192, AES-256）
func NewCrypto() *crypto.Crypto {
	key := os.Getenv("CRYPTO_KEY")
	if key == "" {
		log.Fatal("警告!缺少加密手段,请自负责任!")
	}

	newCrypto, err := crypto.NewCrypto(key)
	if err != nil {
		panic(err)
	}
	return newCrypto
}
