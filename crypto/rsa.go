package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"sync"
)

// RSAWorker rsa算法
type RSAWorker struct {
	locker     sync.Mutex
	workType   RSAType
	signWorker *HashWorker
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// SetPublicKeyFromFile 从文件获取公钥
func (w *RSAWorker) SetPublicKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPublicKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPublicKey 设置base64编码的公钥
func (w *RSAWorker) SetPublicKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	pubKey, err := x509.ParsePKIXPublicKey(bb)
	if err != nil {
		return err
	}
	w.publicKey = pubKey.(*rsa.PublicKey)
	return nil
}

// SetPrivateKeyFromFile 从文件获取私钥
func (w *RSAWorker) SetPrivateKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPrivateKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPrivateKey 设置base64编码的私钥
func (w *RSAWorker) SetPrivateKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	w.privateKey, err = x509.ParsePKCS1PrivateKey(bb)
	if err != nil {
		return err
	}
	return nil
}

// Encode 编码
func (w *RSAWorker) Encode(b []byte) (CValue, error) {
	if w.publicKey == nil {
		return CValue([]byte{}), fmt.Errorf("no publicKey found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	max := w.publicKey.Size() / 2
	buf := bytes.Buffer{}
	var err error
	var res []byte
	for {
		if len(b) <= max {
			res, err = rsa.EncryptPKCS1v15(rand.Reader, w.publicKey, b)
			if err != nil {
				return CValue([]byte{}), err
			}
			buf.Write(res)
			break
		}
		res, err = rsa.EncryptPKCS1v15(rand.Reader, w.publicKey, b[:max])
		if err != nil {
			return CValue([]byte{}), err
		}
		buf.Write(res)
		b = b[max:]
	}
	return CValue(buf.Bytes()), err
}

// Decode 解码
func (w *RSAWorker) Decode(b []byte) (string, error) {
	if w.privateKey == nil {
		return "", fmt.Errorf("no privatekey found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	max := w.privateKey.Size()
	buf := bytes.Buffer{}
	var err error
	var res []byte
	for {
		if len(b) <= max {
			res, err = rsa.DecryptPKCS1v15(rand.Reader, w.privateKey, b)
			if err != nil {
				return "", err
			}
			buf.Write(res)
			break
		}
		res, err = rsa.DecryptPKCS1v15(rand.Reader, w.privateKey, b[:max])
		if err != nil {
			return "", err
		}
		buf.Write(res)
		b = b[max:]
	}
	return buf.String(), err
}

// DecodeBase64 从base64字符串解码
func (w *RSAWorker) DecodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(FillBase64(s))
	if err != nil {
		return "", err
	}
	return w.Decode(b)
}

// Sign 签名，返回签名，hash值
func (w *RSAWorker) Sign(b []byte) (CValue, error) {
	if w.privateKey == nil {
		return CValue([]byte{}), fmt.Errorf("no privatekey found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	signature, err := rsa.SignPSS(rand.Reader, w.privateKey, crypto.SHA256, w.signWorker.Hash(b).Bytes(), nil)
	if err != nil {
		return CValue([]byte{}), err
	}
	return CValue(signature), nil
}

// VerySign 验证签名
func (w *RSAWorker) VerySign(signature, hash []byte) (bool, error) {
	if w.publicKey == nil {
		return false, fmt.Errorf("no publicKey found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	err := rsa.VerifyPSS(w.publicKey, crypto.SHA256, hash, signature, nil)
	return err == nil, nil
}

// VerySignFromBase64 验证base64格式的签名
func (w *RSAWorker) VerySignFromBase64(signature string, hash []byte) (bool, error) {
	if w.publicKey == nil {
		return false, fmt.Errorf("no publicKey found")
	}
	bb, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerySign(bb, hash)
}

// Hash 计算内容的hash
func (w *RSAWorker) Hash(b []byte) CValue {
	return w.signWorker.Hash(b)
}

// NewRSAWorker 创建一个新的rsa算法器
func NewRSAWorker() *RSAWorker {
	w := &RSAWorker{
		locker:     sync.Mutex{},
		workType:   RSA,
		signWorker: NewHashWorker(HashSHA256),
	}
	return w
}
