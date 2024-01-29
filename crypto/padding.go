package crypto

import (
	"bytes"
	"strings"
)

type Padding byte

var (
	NoPadding    Padding = 0
	Pkcs5Padding Padding = 1
	Pkcs7Padding Padding = 2
	ZeroPadding  Padding = 3
)

func noPadding(ciphertext []byte, blockSize int) []byte {
	return ciphertext
}

func noUnpadding(encrypt []byte) []byte {
	return encrypt
}

func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs7Unpadding(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}

func zeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func zeroUnPadding(encrypt []byte) []byte {
	return bytes.TrimFunc(encrypt,
		func(r rune) bool {
			return r == rune(0)
		})
}

// FillBase64 用`=`补全base64长度
func FillBase64(s string) string {
	if x := 4 - len(s)%4; x < 4 {
		return s + strings.Repeat("=", x)
	}
	return s
}
