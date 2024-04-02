package crypto

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"sync"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm4"
	"github.com/tjfoc/gmsm/x509"
)

type SM2 struct {
	locker   sync.Mutex
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
	txt, err = x509.MarshalPKIXPublicKey(&p.PublicKey)
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
	if pubfile != "" {
		txt, _ := x509.WritePublicKeyToPem(w.pubKey)
		err := os.WriteFile(pubfile, txt, 0644)
		if err != nil {
			return err
		}
	}
	if prifile != "" {
		txt, _ := x509.WritePrivateKeyToPem(w.priKey, nil)
		return os.WriteFile(prifile, txt, 0644)
	}
	return nil
}

// SetPublicKeyFromFile 从文件获取公钥
func (w *SM2) SetPublicKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	w.pubKey, err = x509.ReadPublicKeyFromPem(b)
	return err
	// block, _ := pem.Decode(b)
	// return w.SetPublicKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPublicKey 设置base64编码的公钥
func (w *SM2) SetPublicKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	println(key)
	pubKey, err := x509.ParseSm2PublicKey(bb)
	if err != nil {
		println("==========", err.Error())
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
	w.priKey, err = x509.ReadPrivateKeyFromPem(b, nil)
	return err
	// block, _ := pem.Decode(b)
	// return w.SetPrivateKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPrivateKey 设置base64编码的私钥
func (w *SM2) SetPrivateKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	priKey, err := x509.ParsePKCS8UnecryptedPrivateKey(bb)
	if err != nil {
		return err
	}
	w.priKey = priKey
	w.priBytes = bb

	// if len(w.pubBytes) == 0 {
	// 	// 没有载入pubkey，生成新的pubkey
	// 	txt, err := x509.MarshalSm2PublicKey(&priKey.PublicKey)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	w.pubBytes = txt
	// 	w.pubKey = &priKey.PublicKey
	// }
	return nil
}

// Encode sm2加密
func (w *SM2) Encode(b []byte) (CValue, error) {
	if w.pubKey == nil {
		return CValue([]byte{}), fmt.Errorf("no public key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
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
	w.locker.Lock()
	defer w.locker.Unlock()
	c, err := w.priKey.DecryptAsn1(b)
	// c, err := w.priKey.Decrypt(nil, b, nil)
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
	w.locker.Lock()
	defer w.locker.Unlock()
	signature, err := w.priKey.Sign(rand.Reader, b, nil)
	if err != nil {
		return CValue([]byte{}), err
	}
	return CValue(signature), nil
}

// VerifySign 验证签名
func (w *SM2) VerifySign(signature, data []byte) (bool, error) {
	if w.pubKey == nil {
		return false, fmt.Errorf("no public key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	ok := w.pubKey.Verify(data, signature)
	return ok, nil
}

// VerifySignFromBase64 验证base64格式的签名
func (w *SM2) VerifySignFromBase64(signature string, data []byte) (bool, error) {
	bb, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerifySign(bb, data)
}

// VerifySignFromHex 验证hexstring格式的签名
func (w *SM2) VerifySignFromHex(signature string, data []byte) (bool, error) {
	bb, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerifySign(bb, data)
}

// Decrypt 兼容旧方法，直接解析base64字符串
func (w *SM2) Decrypt(s string) string {
	x, _ := w.DecodeBase64(s)
	return x
}

// Encrypt 兼容旧方法，直接返回base64字符串
func (w *SM2) Encrypt(s string) string {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return ""
	}
	return x.Base64String()
}

// EncryptTo 兼容旧方法，直接返回base64字符串
func (w *SM2) EncryptTo(s string) CValue {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return CValue([]byte{})
	}
	return x
}

// NewSM2 创建一个新的sm2算法器
func NewSM2() *SM2 {
	return &SM2{
		locker: sync.Mutex{},
	}
}

// sm4

type SM4 struct {
	locker   sync.Mutex
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

// Decrypt 兼容旧方法，直接解析base64字符串
func (w *SM4) Decrypt(s string) string {
	x, _ := w.DecodeBase64(s)
	return x
}

// Encrypt 兼容旧方法，直接返回base64字符串
func (w *SM4) Encrypt(s string) string {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return ""
	}
	return x.Base64String()
}

// EncryptTo 兼容旧方法，直接返回base64字符串
func (w *SM4) EncryptTo(s string) CValue {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return CValue([]byte{})
	}
	return x
}

// NewSM4 创建一个新的sm4算法器
func NewSM4(t SM4Type) *SM4 {
	return &SM4{
		locker:   sync.Mutex{},
		workType: t,
	}
}
