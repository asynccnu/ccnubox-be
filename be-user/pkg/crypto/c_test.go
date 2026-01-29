package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestDecrypt(t *testing.T) {
	encodedCiphertext := "xxxxxxxx"
	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		fmt.Println(err)
	}

	block, err := aes.NewCipher([]byte("muxiStudioSecret"))
	if err != nil {
		fmt.Println(err)
	}

	if len(ciphertext) < aes.BlockSize {
		fmt.Println("长度不足")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	fmt.Println(string(ciphertext))
}
