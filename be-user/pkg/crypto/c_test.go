package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
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

func TestEncrypt(t *testing.T) {
	plaintext := "xxxxxxxx"

	block, err := aes.NewCipher([]byte("muxiStudioSecret"))
	if err != nil {
		fmt.Println(err)
		return
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))

	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		fmt.Println(err)
		return
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))

	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	fmt.Println(encoded)
}
