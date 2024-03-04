package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"sync"
)

type RSABits byte

var (
	RSA2048 RSABits = 1
	RSA4096 RSABits = 2
)

// RSA rsa算法
type RSA struct {
	locker   sync.Mutex
	signHash *HASH
	pubKey   *rsa.PublicKey
	priKey   *rsa.PrivateKey
	pubBytes CValue
	priBytes CValue
}

// Keys 返回公钥和私钥
func (w *RSA) Keys() (CValue, CValue) {
	return w.pubBytes, w.priBytes
}

// GenerateKey 创建ecc密钥对
//
//	返回，pubkey，prikey，error
func (w *RSA) GenerateKey(bits RSABits) (CValue, CValue, error) {
	var p *rsa.PrivateKey
	switch bits {
	case RSA2048:
		p, _ = rsa.GenerateKey(rand.Reader, 2048)
	case RSA4096:
		p, _ = rsa.GenerateKey(rand.Reader, 4096)
	}
	txt := x509.MarshalPKCS1PrivateKey(p)
	w.priBytes = txt
	txt, err := x509.MarshalPKIXPublicKey(&p.PublicKey)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	w.pubBytes = txt
	w.pubKey = &p.PublicKey
	w.priKey = p
	return w.pubBytes, w.priBytes, nil
}

// ToFile 创建ecc密钥到文件
func (w *RSA) ToFile(pubfile, prifile string) error {
	block := &pem.Block{
		Type:  "rsa public key",
		Bytes: w.pubBytes.Bytes(),
	}
	txt := pem.EncodeToMemory(block)
	err := os.WriteFile(pubfile, txt, 0644)
	if err != nil {
		return err
	}
	block = &pem.Block{
		Type:  "rsa public key",
		Bytes: w.priBytes.Bytes(),
	}
	txt = pem.EncodeToMemory(block)
	err = os.WriteFile(prifile, txt, 0644)
	return err
}

// SetPublicKeyFromFile 从文件获取公钥
func (w *RSA) SetPublicKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPublicKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPublicKey 设置base64编码的公钥
func (w *RSA) SetPublicKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	pubKey, err := x509.ParsePKIXPublicKey(bb)
	if err != nil {
		return err
	}
	w.pubBytes = bb
	w.pubKey = pubKey.(*rsa.PublicKey)
	return nil
}

// SetPrivateKeyFromFile 从文件获取私钥
func (w *RSA) SetPrivateKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPrivateKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPrivateKey 设置base64编码的私钥
func (w *RSA) SetPrivateKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	w.priKey, err = x509.ParsePKCS1PrivateKey(bb)
	if err != nil {
		return err
	}
	w.priBytes = bb

	if len(w.pubBytes) == 0 {
		// 没有载入国pubkey，生成新的pubkey
		txt, err := x509.MarshalPKIXPublicKey(&w.priKey.PublicKey)
		if err != nil {
			return err
		}
		w.pubBytes = txt
		w.pubKey = &w.priKey.PublicKey
	}
	return nil
}

// Encode 编码
func (w *RSA) Encode(b []byte) (CValue, error) {
	if w.pubKey == nil {
		return CValue([]byte{}), fmt.Errorf("no public key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	max := w.pubKey.Size() / 2
	buf := bytes.Buffer{}
	var err error
	var res []byte
	for {
		if len(b) <= max {
			res, err = rsa.EncryptPKCS1v15(rand.Reader, w.pubKey, b)
			if err != nil {
				return CValue([]byte{}), err
			}
			buf.Write(res)
			break
		}
		res, err = rsa.EncryptPKCS1v15(rand.Reader, w.pubKey, b[:max])
		if err != nil {
			return CValue([]byte{}), err
		}
		buf.Write(res)
		b = b[max:]
	}
	return CValue(buf.Bytes()), err
}

// Decode 解码
func (w *RSA) Decode(b []byte) (string, error) {
	if w.priKey == nil {
		return "", fmt.Errorf("no private key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	max := w.priKey.Size()
	buf := bytes.Buffer{}
	var err error
	var res []byte
	for {
		if len(b) <= max {
			res, err = rsa.DecryptPKCS1v15(rand.Reader, w.priKey, b)
			if err != nil {
				return "", err
			}
			buf.Write(res)
			break
		}
		res, err = rsa.DecryptPKCS1v15(rand.Reader, w.priKey, b[:max])
		if err != nil {
			return "", err
		}
		buf.Write(res)
		b = b[max:]
	}
	return buf.String(), err
}

// DecodeBase64 从base64字符串解码
func (w *RSA) DecodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(FillBase64(s))
	if err != nil {
		return "", err
	}
	return w.Decode(b)
}

// Sign 签名，返回签名，hash值
func (w *RSA) Sign(b []byte) (CValue, error) {
	if w.priKey == nil {
		return CValue([]byte{}), fmt.Errorf("no private key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	signature, err := rsa.SignPSS(rand.Reader, w.priKey, crypto.SHA256, w.signHash.Hash(b).Bytes(), nil)
	if err != nil {
		return CValue([]byte{}), err
	}
	return CValue(signature), nil
}

// VerySign 验证签名
func (w *RSA) VerySign(signature, data []byte) (bool, error) {
	if w.pubKey == nil {
		return false, fmt.Errorf("no public key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	err := rsa.VerifyPSS(w.pubKey, crypto.SHA256, w.signHash.Hash(data).Bytes(), signature, nil)
	return err == nil, nil
}

// VerySignFromBase64 验证base64格式的签名
func (w *RSA) VerySignFromBase64(signature string, data []byte) (bool, error) {
	bb, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerySign(bb, data)
}

// VerySignFromHex 验证hexstring格式的签名
func (w *RSA) VerySignFromHex(signature string, data []byte) (bool, error) {
	bb, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerySign(bb, data)
}

// Decrypt 兼容旧方法，直接解析base64字符串
func (w *RSA) Decrypt(s string) string {
	x, _ := w.DecodeBase64(s)
	return x
}

// Encrypt 兼容旧方法，直接返回base64字符串
func (w *RSA) Encrypt(s string) string {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return ""
	}
	return x.Base64String()
}

// EncryptTo 兼容旧方法，直接返回base64字符串
func (w *RSA) EncryptTo(s string) CValue {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return CValue([]byte{})
	}
	return x
}

// NewRSA 创建一个新的rsa算法器
//
//	签名算法采用sha256
func NewRSA() *RSA {
	w := &RSA{
		locker:   sync.Mutex{},
		signHash: NewHash(HashSHA256),
	}
	return w
}
