package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto/ecies"
)

type ECShortName byte

var (
	// ECPrime256v1 as elliptic.P256() and openssl ecparam -name prime256v1
	ECPrime256v1 ECShortName = 1
	// ECSecp384r1 as elliptic.P384() and openssl ecparam -name secp384r1
	ECSecp384r1 ECShortName = 2
)

// ECC ecc算法
type ECC struct {
	locker   sync.Mutex
	signHash *HASH
	pubKey   *ecdsa.PublicKey
	priKey   *ecdsa.PrivateKey
	pubEcies *ecies.PublicKey
	priEcies *ecies.PrivateKey
	pubBytes CValue
	priBytes CValue
}

// Keys 返回公钥和私钥
func (w *ECC) Keys() (CValue, CValue) {
	return w.pubBytes, w.priBytes
}

// GenerateKey 创建ecc密钥对
//
//	返回，pubkey，prikey，error
func (w *ECC) GenerateKey(ec ECShortName) (CValue, CValue, error) {
	var p *ecdsa.PrivateKey
	switch ec {
	case ECPrime256v1:
		p, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case ECSecp384r1:
		p, _ = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	}
	txt, err := x509.MarshalECPrivateKey(p)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	w.priBytes = txt
	txt, err = x509.MarshalPKIXPublicKey(&p.PublicKey)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	w.pubBytes = txt
	w.pubKey = &p.PublicKey
	w.pubEcies = ecies.ImportECDSAPublic(w.pubKey)
	w.priKey = p
	w.priEcies = ecies.ImportECDSA(p)
	return w.pubBytes, w.priBytes, nil
}

// ToFile 创建ecc密钥到文件
func (w *ECC) ToFile(pubfile, prifile string) error {
	if pubfile != "" {
		block := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: w.pubBytes.Bytes(),
		}
		txt := pem.EncodeToMemory(block)
		err := os.WriteFile(pubfile, txt, 0644)
		if err != nil {
			return err
		}
	}
	if prifile != "" {
		block := &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: w.priBytes.Bytes(),
		}
		txt := pem.EncodeToMemory(block)
		return os.WriteFile(prifile, txt, 0644)
	}
	return nil
}

// SetPublicKeyFromFile 从文件获取公钥
func (w *ECC) SetPublicKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPublicKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPublicKey 设置base64编码的公钥
func (w *ECC) SetPublicKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	pubKey, err := x509.ParsePKIXPublicKey(bb)
	if err != nil {
		return err
	}
	w.pubKey = pubKey.(*ecdsa.PublicKey)
	w.pubEcies = ecies.ImportECDSAPublic(w.pubKey)
	w.pubBytes = bb
	return nil
}

// SetPrivateKeyFromFile 从文件获取私钥
func (w *ECC) SetPrivateKeyFromFile(keyPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(b)
	return w.SetPrivateKey(base64.StdEncoding.EncodeToString(block.Bytes))
}

// SetPrivateKey 设置base64编码的私钥
func (w *ECC) SetPrivateKey(key string) error {
	bb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	priKey, err := x509.ParseECPrivateKey(bb)
	if err != nil {
		if strings.Contains(err.Error(), "use ParsePKCS8PrivateKey instead") {
			priKeypk8, err := x509.ParsePKCS8PrivateKey(bb)
			if err != nil {
				return err
			}
			priKey = priKeypk8.(*ecdsa.PrivateKey)
		} else {
			return err
		}
	}
	w.priKey = priKey
	w.priEcies = ecies.ImportECDSA(priKey)
	w.priBytes = bb

	if len(w.pubBytes) == 0 {
		// 没有载入国pubkey，生成新的pubkey
		txt, err := x509.MarshalPKIXPublicKey(&priKey.PublicKey)
		if err != nil {
			return err
		}
		w.pubBytes = txt
		w.pubKey = &priKey.PublicKey
		w.pubEcies = ecies.ImportECDSAPublic(w.pubKey)
	}
	return nil
}

// Encode ecc加密
func (w *ECC) Encode(b []byte) (CValue, error) {
	if w.pubEcies == nil {
		return CValue([]byte{}), fmt.Errorf("no public key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	res, err := ecies.Encrypt(rand.Reader, w.pubEcies, b, nil, nil)
	if err != nil {
		return CValue([]byte{}), err
	}
	return CValue(res), nil
}

// Decode ecc解密
func (w *ECC) Decode(b []byte) (string, error) {
	if w.priEcies == nil {
		return "", fmt.Errorf("no private key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	c, err := w.priEcies.Decrypt(b, nil, nil)
	if err != nil {
		return "", err
	}
	return String(c), nil
}

// DecodeBase64 从base64字符串解码
func (w *ECC) DecodeBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(FillBase64(s))
	if err != nil {
		return "", err
	}
	return w.Decode(b)
}

// Sign 签名
func (w *ECC) Sign(b []byte) (CValue, error) {
	if w.priKey == nil {
		return CValue([]byte{}), fmt.Errorf("no private key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	signature, err := ecdsa.SignASN1(rand.Reader, w.priKey, w.signHash.Hash(b).Bytes())
	if err != nil {
		return CValue([]byte{}), err
	}
	return CValue(signature), nil
}

// VerifySign 验证签名
func (w *ECC) VerifySign(signature, data []byte) (bool, error) {
	if w.pubKey == nil {
		return false, fmt.Errorf("no public key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	ok := ecdsa.VerifyASN1(w.pubKey, w.signHash.Hash(data).Bytes(), signature)
	return ok, nil
}

// VerifySignFromBase64 验证base64格式的签名
func (w *ECC) VerifySignFromBase64(signature string, data []byte) (bool, error) {
	bb, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerifySign(bb, data)
}

// VerifySignFromHex 验证hexstring格式的签名
func (w *ECC) VerifySignFromHex(signature string, data []byte) (bool, error) {
	bb, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerifySign(bb, data)
}

// Decrypt 兼容旧方法，直接解析base64字符串
func (w *ECC) Decrypt(s string) string {
	x, _ := w.DecodeBase64(s)
	return x
}

// Encrypt 加密，兼容旧方法，直接返回base64字符串
func (w *ECC) Encrypt(s string) string {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return ""
	}
	return x.Base64String()
}

// EncryptTo 加密字符串
func (w *ECC) EncryptTo(s string) CValue {
	x, err := w.Encode(Bytes(s))
	if err != nil {
		return CValue([]byte{})
	}
	return x
}

// NewECC 创建一个新的ecc算法器
//
//	签名算法采用sha256
//	支持 openssl ecparam -name prime256v1/secp384r1 格式的密钥
func NewECC() *ECC {
	w := &ECC{
		locker:   sync.Mutex{},
		signHash: NewHash(HashSHA256),
	}
	return w
}
