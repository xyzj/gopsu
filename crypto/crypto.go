// Package crypto 加密解密
package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"strings"
	"unsafe"
)

type HashType byte
type AESType byte
type RSAType byte

// CValue 加密后的数据，可输出[]byte,hex string,base64string
type CValue []byte

// Len 加密结果长度
func (v CValue) Len() int {
	return len(v)
}

// Bytes 加密结果
func (v CValue) Bytes() []byte {
	return v
}

// HexString 加密结果以hex字符串形式输出
func (v CValue) HexString() string {
	return hex.EncodeToString(v)
}

// Base64String 加密结果以标准base64字符串形式输出
func (v CValue) Base64String() string {
	return base64.StdEncoding.EncodeToString(v)
}

// Base64StringNoTail 加密结果以标准base64字符串形式输出，去除`=`
func (v CValue) Base64StringNoTail() string {
	return strings.ReplaceAll(base64.StdEncoding.EncodeToString(v), "=", "")
}

// URLBase64String 加密结果以URLbase64字符串形式输出
func (v CValue) URLBase64String() string {
	return base64.URLEncoding.EncodeToString(v)
}

const (
	// HashMD5 md5算法
	HashMD5 HashType = 1
	// HashSHA256 sha256算法
	HashSHA256 HashType = 2
	// HashSHA512 sha512算法
	HashSHA512 HashType = 3
	// HashHMACSHA1 hmacsha1摘要算法
	HashHMACSHA1 HashType = 4
	// HashHMACSHA256 hmacsha256摘要算法
	HashHMACSHA256 HashType = 5
	// HashSHA1 sha1算法
	HashSHA1 HashType = 6
	// AES128CBC aes128cbc算法
	AES128CBC AESType = 1
	// AES128CFB aes128cfb算法
	AES128CFB AESType = 2
	// AES192CBC aes192cbc算法
	AES192CBC AESType = 3
	// AES192CFB aes192cfb算法
	AES192CFB AESType = 4
	// AES256CBC aes256cbc算法
	AES256CBC AESType = 5
	// AES256CFB aes256cfb算法
	AES256CFB AESType = 6
	// RSA rsa加密解密
	RSA RSAType = 1
)

var (
	md5hash    = NewHashWorker(HashMD5)
	sha1hash   = NewHashWorker(HashSHA1)
	sha256hash = NewHashWorker(HashSHA256)
	sha512hash = NewHashWorker(HashSHA512)
)

// String 内存地址转换[]byte
func String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Bytes 内存地址转换string
func Bytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			cap int
		}{string: s, cap: len(s)},
	))
}

// GetMD5 生成md5字符串
func GetMD5(text string) string {
	return md5hash.Hash(Bytes(text)).HexString()
}

// GetSHA1 生成sha1字符串
func GetSHA1(text string) string {
	return sha1hash.Hash(Bytes(text)).HexString()
}

// GetSHA256 生成sha256字符串
func GetSHA256(text string) string {
	return sha256hash.Hash(Bytes(text)).HexString()
}

// GetSHA512 生成sha512字符串
func GetSHA512(text string) string {
	return sha512hash.Hash(Bytes(text)).HexString()
}

// GetRandom 获取随机数据
func GetRandom(l int) []byte {
	buf := make([]byte, l)
	io.ReadFull(rand.Reader, buf)
	return buf
}
