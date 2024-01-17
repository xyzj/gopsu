package crypto

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"sync"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm4"
	"github.com/tjfoc/gmsm/x509"
)

type SM2 struct {
	sync.Mutex
	pubKey   *sm2.PublicKey
	priKey   *sm2.PrivateKey
	pubBytes CValue
	priBytes CValue
}

// Keys 返回公钥和私钥
func (w *SM2) Keys() (CValue, CValue) {
	return w.pubBytes, w.priBytes
}

// GenerateKey 创建sm2密钥对
//
//	返回，pubkey，prikey，error
func (w *SM2) GenerateKey() (CValue, CValue, error) {
	p, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	txt, err := x509.MarshalSm2UnecryptedPrivateKey(p)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	w.priBytes = txt
	txt, err = x509.MarshalSm2PublicKey(&p.PublicKey)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	w.pubBytes = txt
	w.priKey = p
	w.pubKey = &p.PublicKey
	return w.pubBytes, w.priBytes, nil
}

// ToFile 创建ecc密钥到文件
func (w *SM2) ToFile(pubfile, prifile string) error {
	block := &pem.Block{
		Type:  "sm2 public key",
		Bytes: w.pubBytes.Bytes(),
	}
	txt := pem.EncodeToMemory(block)
	err := os.WriteFile(pubfile, txt, 0644)
	if err != nil {
		return err
	}
	block = &pem.Block{
		Type:  "sm2 public key",
		Bytes: w.priBytes.Bytes(),
	}
	txt = pem.EncodeToMemory(block)
	err = os.WriteFile(prifile, txt, 0644)
	return err
}

// SetPublicKeyFromFile 从文件获取公钥
func (w *SM2) SetPublicKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPublicKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPublicKey 设置base64编码的公钥
func (w *SM2) SetPublicKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	pubKey, err := x509.ParseSm2PublicKey(bb)
	if err != nil {
		return err
	}
	w.pubKey = pubKey
	w.pubBytes = bb
	return nil
}

// SetPrivateKeyFromFile 从文件获取私钥
func (w *SM2) SetPrivateKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPrivateKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPrivateKey 设置base64编码的私钥
func (w *SM2) SetPrivateKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	priKey, err := x509.ParseSm2PrivateKey(bb)
	if err != nil {
		return err
	}
	w.priKey = priKey
	w.priBytes = bb
	return nil
}

// Encode sm2加密
func (w *SM2) Encode(b []byte) (CValue, error) {
	if w.pubKey == nil {
		return CValue([]byte{}), fmt.Errorf("no public key found")
	}
	w.Lock()
	defer w.Unlock()
	res, err := w.pubKey.EncryptAsn1(b, rand.Reader)
	if err != nil {
		return CValue([]byte{}), err
	}
	return CValue(res), nil
}

// Decode sm2解密
func (w *SM2) Decode(b []byte) (string, error) {
	if w.priKey == nil {
		return "", fmt.Errorf("no private key found")
	}
	w.Lock()
	defer w.Unlock()
	c, err := w.priKey.DecryptAsn1(b)
	if err != nil {
		return "", err
	}
	return String(c), nil
}

// DecodeBase64 从base64字符串解码
func (w *SM2) DecodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(FillBase64(s))
	if err != nil {
		return "", err
	}
	return w.Decode(b)
}

// Sign 签名
func (w *SM2) Sign(b []byte) (CValue, error) {
	if w.priKey == nil {
		return CValue([]byte{}), fmt.Errorf("no private key found")
	}
	w.Lock()
	defer w.Unlock()
	signature, err := w.priKey.Sign(rand.Reader, b, nil)
	if err != nil {
		return CValue([]byte{}), err
	}
	return CValue(signature), nil
}

// VerySign 验证签名
func (w *SM2) VerySign(signature, data []byte) (bool, error) {
	if w.pubKey == nil {
		return false, fmt.Errorf("no public key found")
	}
	w.Lock()
	defer w.Unlock()
	ok := w.pubKey.Verify(data, signature)
	return ok, nil
}

// VerySignFromBase64 验证base64格式的签名
func (w *SM2) VerySignFromBase64(signature string, data []byte) (bool, error) {
	bb, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerySign(bb, data)
}

// VerySignFromHex 验证hexstring格式的签名
func (w *SM2) VerySignFromHex(signature string, data []byte) (bool, error) {
	bb, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerySign(bb, data)
}

// NewSM2 创建一个新的sm2算法器
func NewSM2() *SM2 {
	return &SM2{
		Mutex: sync.Mutex{},
	}
}

// sm4

type SM4 struct {
	sync.Mutex
	workType SM4Type
	key      []byte
	iv       []byte
	appendiv bool
}

const SM4BlockSize = 16

// SetKeyIV 设置iv和key
// 如果不设置iv，会生成随机iv并追加在加密结果的头部
func (w *SM4) SetKeyIV(key, iv []byte) error {
	if iv == nil {
		w.appendiv = true
		iv = GetRandom(aes.BlockSize)
	}
	if len(key) < SM4BlockSize || len(iv) < SM4BlockSize {
		return fmt.Errorf("key length must be longer than %d, and the length of iv must be longer than %d", SM4BlockSize, SM4BlockSize)
	}
	w.key = key[:SM4BlockSize]
	w.iv = iv[:SM4BlockSize]
	return sm4.SetIV(w.iv)
}

// Encode sm4加密
func (w *SM4) Encode(b []byte) (CValue, error) {
	switch w.workType {
	case SM4CBC:
		return sm4.Sm4Cbc(w.key, b, true)
	case SM4CFB:
		return sm4.Sm4CFB(w.key, b, true)
	case SM4ECB:
		return sm4.Sm4Ecb(w.key, b, true)
	case SM4OFB:
		return sm4.Sm4OFB(w.key, b, true)
	}
	return CValue([]byte{}), nil
}

// Decode sm4解密
func (w *SM4) Decode(b []byte) (string, error) {
	var bb []byte
	var err error
	switch w.workType {
	case SM4CBC:
		bb, err = sm4.Sm4Cbc(w.key, b, false)
	case SM4CFB:
		bb, err = sm4.Sm4CFB(w.key, b, false)
	case SM4ECB:
		bb, err = sm4.Sm4Ecb(w.key, b, false)
	case SM4OFB:
		bb, err = sm4.Sm4OFB(w.key, b, false)
	}
	return String(bb), err
}

// DecodeBase64 解密base64编码的字符串
func (w *SM4) DecodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(FillBase64(s))
	if err != nil {
		return "", err
	}
	return w.Decode(b)
}

// NewSM4 创建一个新的sm4算法器
func NewSM4(t SM4Type) *SM4 {
	return &SM4{
		Mutex:    sync.Mutex{},
		workType: t,
	}
}
