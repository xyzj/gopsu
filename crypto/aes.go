package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"sync"
)

// AESWorker aes算法
type AESWorker struct {
	locker   sync.Mutex
	workType AESType
	block    cipher.Block
	iv       []byte
	padding  bool
	appendiv bool
}

// EnableCFBPadding 对cfb的数据进行pkcspadding
func (w *AESWorker) EnableCFBPadding() {
	w.padding = true
}

// SetKeyIV 设置iv和key
// 如果不设置iv，会生成随机iv并追加在加密结果的头部
func (w *AESWorker) SetKeyIV(key, iv []byte) error {
	var l = 16
	switch w.workType {
	case AES192CBC, AES192CFB:
		l = 24
	case AES256CBC, AES256CFB:
		l = 32
	}
	if iv == nil {
		w.appendiv = true
		iv = GetRandom(aes.BlockSize)
	}
	if len(key) < l || len(iv) < aes.BlockSize {
		return fmt.Errorf("key length must be longer than %d, and the length of iv must be longer than %d", l, aes.BlockSize)
	}
	w.block, _ = aes.NewCipher(key[:l])
	w.iv = iv[:aes.BlockSize]
	return nil
}

// Encode aes加密
func (w *AESWorker) Encode(b []byte) (CValue, error) {
	if w.block == nil {
		return CValue([]byte{}), fmt.Errorf("key or iv are not set")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	var content = b
	if w.padding {
		content = pkcs7Padding(b, aes.BlockSize)
	}
	var crypted []byte
	var idx = 0
	if w.appendiv {
		crypted = make([]byte, aes.BlockSize+len(content))
		copy(crypted, w.iv)
		idx = aes.BlockSize
	} else {
		crypted = make([]byte, len(content))
	}
	// 不能预初始化，否则二次编码过程会出错
	switch w.workType {
	case AES128CBC, AES192CBC, AES256CBC:
		cipher.NewCBCEncrypter(w.block, w.iv).CryptBlocks(crypted[idx:], content)
	case AES128CFB, AES192CFB, AES256CFB:
		cipher.NewCFBEncrypter(w.block, w.iv).XORKeyStream(crypted[idx:], content)
	}
	return CValue(crypted), nil
}

// Decode aes解密
func (w *AESWorker) Decode(b []byte) (string, error) {
	if w.block == nil {
		return "", fmt.Errorf("key or iv are not set")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	if w.appendiv {
		w.iv = b[:aes.BlockSize]
		b = b[aes.BlockSize:]
	}
	switch w.workType {
	case AES128CBC, AES192CBC, AES256CBC:
		decrypted := make([]byte, len(b))
		cipher.NewCBCDecrypter(w.block, w.iv).CryptBlocks(decrypted, b)
		return String(pkcs7Unpadding(decrypted)), nil
	case AES128CFB, AES192CFB, AES256CFB:
		cipher.NewCFBDecrypter(w.block, w.iv).XORKeyStream(b, b)
		if w.padding {
			return String(pkcs7Unpadding(b)), nil
		}
		return String(b), nil
	}
	return "", fmt.Errorf("unsupport cipher type")
}

// DecodeBase64 aes解密base64编码的字符串
func (w *AESWorker) DecodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(FillBase64(s))
	if err != nil {
		return "", err
	}
	return w.Decode(b)
}

// NewAESWorker 创建一个新的aes加密解密器
func NewAESWorker(t AESType) *AESWorker {
	w := &AESWorker{
		locker:   sync.Mutex{},
		workType: t,
	}
	switch t {
	case AES128CBC, AES192CBC, AES256CBC:
		w.padding = true
	}
	return w
}
