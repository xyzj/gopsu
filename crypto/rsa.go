package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"sort"
	"sync"
	"time"
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

// GenerateKey 创建rsa密钥对
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
	txt, err := x509.MarshalPKCS8PrivateKey(p)
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
	w.priKey = p
	return w.pubBytes, w.priBytes, nil
}

// ToFile 创建rsa密钥到文件
func (w *RSA) ToFile(pubfile, prifile string) error {
	if pubfile != "" {
		block := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: w.pubBytes.Bytes(),
		}
		txt := pem.EncodeToMemory(block)
		return os.WriteFile(pubfile, txt, 0o644)
	}
	if prifile != "" {
		block := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: w.priBytes.Bytes(),
		}
		txt := pem.EncodeToMemory(block)
		return os.WriteFile(prifile, txt, 0o644)
	}
	return nil
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
	priKey, err := x509.ParsePKCS8PrivateKey(bb)
	if err != nil {
		return err
	}
	w.priBytes = bb
	w.priKey = priKey.(*rsa.PrivateKey)

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
		return EmptyValue, fmt.Errorf("no public key found")
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
				return EmptyValue, err
			}
			buf.Write(res)
			break
		}
		res, err = rsa.EncryptPKCS1v15(rand.Reader, w.pubKey, b[:max])
		if err != nil {
			return EmptyValue, err
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
		return EmptyValue, fmt.Errorf("no private key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	signature, err := rsa.SignPSS(rand.Reader, w.priKey, crypto.SHA256, w.signHash.Hash(b).Bytes(), nil)
	if err != nil {
		return EmptyValue, err
	}
	return CValue(signature), nil
}

// VerifySign 验证签名
func (w *RSA) VerifySign(signature, data []byte) (bool, error) {
	if w.pubKey == nil {
		return false, fmt.Errorf("no public key found")
	}
	w.locker.Lock()
	defer w.locker.Unlock()
	err := rsa.VerifyPSS(w.pubKey, crypto.SHA256, w.signHash.Hash(data).Bytes(), signature, nil)
	return err == nil, nil
}

// VerifySignFromBase64 验证base64格式的签名
func (w *RSA) VerifySignFromBase64(signature string, data []byte) (bool, error) {
	bb, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerifySign(bb, data)
}

// VerifySignFromHex 验证hexstring格式的签名
func (w *RSA) VerifySignFromHex(signature string, data []byte) (bool, error) {
	bb, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return w.VerifySign(bb, data)
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
		return EmptyValue
	}
	return x
}

// CreateCert 创建基于rsa算法的数字证书，opt.RootKey无效时，会重新创建私钥和根证书
func (w *RSA) CreateCert(opt *CertOpt) error {
	// 处理参数
	if opt == nil {
		opt = &CertOpt{
			DNS: []string{},
			IP:  []string{},
		}
	}
	if len(opt.DNS) == 0 {
		opt.DNS = []string{"localhost"}
	}
	if len(opt.IP) == 0 {
		opt.IP = []string{"127.0.0.1"}
	}
	ips := make([]net.IP, 0, len(opt.IP))
	sort.Slice(opt.IP, func(i, j int) bool {
		return opt.IP[i] < opt.IP[j]
	})
	for _, v := range opt.IP {
		ips = append(ips, net.ParseIP(v))
	}
	// 处理根证书
	var rootDer, txt []byte
	var err error
	var rootCsr *x509.Certificate
	// 检查私钥
	if opt.RootKey != "" {
		w.SetPrivateKeyFromFile(opt.RootKey)
	}
	if w.priKey == nil {
		opt.RootCa = ""
		opt.RootKey = ""
		w.GenerateKey(RSA2048)
	}
	// 创建根证书
	if opt.RootCa != "" {
		b, err := os.ReadFile(opt.RootCa)
		if err == nil {
			p, _ := pem.Decode(b)
			rootCsr, err = x509.ParseCertificate(p.Bytes)
			if err != nil {
				return err
			}
		}
	}
	if rootCsr == nil {
		rootCsr = &x509.Certificate{
			Version:      3,
			SerialNumber: big.NewInt(time.Now().Unix()),
			Subject: pkix.Name{
				Country:    []string{"CN"},
				Province:   []string{"Shanghai"},
				Locality:   []string{"Shanghai"},
				CommonName: "xyzj",
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(68, 0, 0),
			MaxPathLen:            1,
			BasicConstraintsValid: true,
			IsCA:                  true,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
			// ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		}
	}
	rootDer, err = x509.CreateCertificate(rand.Reader, rootCsr, rootCsr, w.pubKey, w.priKey)
	if err != nil {
		return err
	}
	// 创建服务器证书
	certCsr := &x509.Certificate{
		Version:      3,
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			Country:    []string{"CN"},
			Province:   []string{"Shanghai"},
			Locality:   []string{"Shanghai"},
			CommonName: "xyzj",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(68, 0, 0),
		DNSNames:    opt.DNS,
		IPAddresses: ips,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	// 创建网站私钥
	p, _ := rsa.GenerateKey(rand.Reader, 2048)
	certDer, err := x509.CreateCertificate(rand.Reader, certCsr, rootCsr, &p.PublicKey, w.priKey)
	// certDer, err := x509.CreateCertificate(rand.Reader, certCsr, rootCsr, w.pubKey, w.priKey)
	if err != nil {
		return err
	}
	// 保存网站证书
	txt = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDer,
	})
	err = os.WriteFile("cert.rsa.pem", txt, 0o664)
	if err != nil {
		return err
	}
	// 保存网站私钥
	txt, err = x509.MarshalPKCS8PrivateKey(p)
	if err != nil {
		return err
	}
	txt = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: txt,
	})
	err = os.WriteFile("cert-key.rsa.pem", txt, 0o664)
	if err != nil {
		return err
	}
	// 保存root私钥
	if opt.RootKey == "" {
		// 保存根证书
		txt = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: rootDer,
		})
		err = os.WriteFile("root.rsa.pem", txt, 0o664)
		if err != nil {
			return err
		}
		w.ToFile("", "root-key.rsa.pem")
	}
	return nil
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
