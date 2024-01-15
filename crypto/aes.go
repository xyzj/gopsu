package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
)

// AESWorker aes算法
type AESWorker struct {
	locker   sync.Mutex
	cbce     cipher.BlockMode
	cbcd     cipher.BlockMode
	cfbe     cipher.Stream
	cfbd     cipher.Stream
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
		iv = make([]byte, aes.BlockSize)
		io.ReadFull(rand.Reader, iv)
	}
	if len(key) < l || len(iv) < aes.BlockSize {
		return fmt.Errorf("key length must be longer than %d, and the length of iv must be longer than %d", l, aes.BlockSize)
	}
	switch w.workType {
	case AES128CBC, AES192CBC, AES256CBC:
		w.block, _ = aes.NewCipher([]byte(key)[:l])
		w.iv = []byte(iv)[:aes.BlockSize]
		w.cbce = cipher.NewCBCEncrypter(w.block, w.iv)
		w.cbcd = cipher.NewCBCDecrypter(w.block, w.iv)
	case AES128CFB, AES192CFB, AES256CFB:
		w.block, _ = aes.NewCipher([]byte(key)[:l])
		w.iv = []byte(iv)[:aes.BlockSize]
		w.cfbe = cipher.NewCFBEncrypter(w.block, w.iv)
		w.cfbd = cipher.NewCFBDecrypter(w.block, w.iv)
	}
	return nil
}

// Encode aes加密
func (w *AESWorker) Encode(b []byte) (CValue, error) {
	w.locker.Lock()
	defer w.locker.Unlock()
	switch w.workType {
	case AES128CBC, AES192CBC, AES256CBC:
		if w.cbce == nil {
			return CValue([]byte{}), fmt.Errorf("key or iv are not set")
		}
		content := pkcs7Padding(b, aes.BlockSize)
		var crypted []byte
		if w.appendiv {
			crypted = make([]byte, aes.BlockSize+len(content))
			copy(crypted, w.iv)
			w.cbce.CryptBlocks(crypted[aes.BlockSize:], content)
		} else {
			crypted = make([]byte, len(content))
			w.cbce.CryptBlocks(crypted, content)
		}
		return CValue(crypted), nil
	case AES128CFB, AES192CFB, AES256CFB:
		if w.cfbe == nil {
			return CValue([]byte{}), fmt.Errorf("key or iv are not set")
		}
		var content = b
		if w.padding {
			content = pkcs7Padding(b, aes.BlockSize)
		}
		var crypted []byte
		if w.appendiv {
			crypted = make([]byte, aes.BlockSize+len(content))
			copy(crypted, w.iv)
			w.cfbe.XORKeyStream(crypted[aes.BlockSize:], content)
		} else {
			crypted = make([]byte, len(content))
			w.cfbe.XORKeyStream(crypted, content)
		}
		return CValue(crypted), nil
	}
	return CValue([]byte{}), nil
}

// Decode aes解密
func (w *AESWorker) Decode(b []byte) (string, error) {
	w.locker.Lock()
	defer w.locker.Unlock()
	switch w.workType {
	case AES128CBC, AES192CBC, AES256CBC:
		if w.cbce == nil {
			return "", fmt.Errorf("key or iv are not set")
		}
		if w.appendiv {
			w.iv = b[:aes.BlockSize]
			w.cbcd = cipher.NewCBCDecrypter(w.block, w.iv)
			b = b[aes.BlockSize:]
		}
		decrypted := make([]byte, len(b))
		w.cbcd.CryptBlocks(decrypted, b)
		return String(pkcs7Unpadding(decrypted)), nil
	case AES128CFB, AES192CFB, AES256CFB:
		if w.cfbd == nil {
			return "", fmt.Errorf("key or iv are not set")
		}
		if w.appendiv {
			w.iv = b[:aes.BlockSize]
			w.cfbd = cipher.NewCFBDecrypter(w.block, w.iv)
			b = b[aes.BlockSize:]
		}
		if len(b) < aes.BlockSize {
			return "", fmt.Errorf("ciphertext too short")
		}
		w.cfbd.XORKeyStream(b, b)
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
	return w
}
