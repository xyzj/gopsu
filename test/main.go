package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/json"
)

var (
	version     = "0.0.0"
	goVersion   = ""
	buildDate   = ""
	platform    = ""
	author      = "Xu Yuan"
	programName = "Asset Data Center"
)

// 结构定义
// 设备型号信息
type devmod struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Sys    string `json:"-"`
	Remark string `json:"remark,omitempty"`
	pinyin string
}

func (d devmod) DoNoting() {
}

type BaseMap struct {
	sync.RWMutex
	data map[string]string
}

func FormatMQBody(d []byte) string {
	if json.Valid(d) {
		return gopsu.String(d)
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, gopsu.String(d))
	// return base64.StdEncoding.EncodeToString(d)
}

func test(a bool, b ...string) {
	if len(b) == 0 {
		println("no b")
	} else {
		if b[0] == "" {
			println("nadadadf")
		} else {
			println("123123123")
		}
	}
	if a {
		defer println("defer")
	}
	println("done")
}

var (
	conf  = flag.String("conf", "", "usage")
	conf1 = flag.String("conf1", "", "usage")
	conf2 = flag.String("conf2", "", "usage")
)

func mqttcb(topic string, body []byte) {
	println("---", topic, string(body))
}

type aaa struct {
	Username string           `json:"username" yaml:"username"`
	Password config.PwdString `json:"pwd" yaml:"pwd"`
}

type serviceParams struct {
	Params     []string `yaml:"params"`
	Exec       string   `yaml:"exec"`
	Enable     bool     `yaml:"enable"`
	manualStop bool     `yaml:"-"`
}

func RSAGenKey(bits int) error {
	/*
		生成私钥
	*/
	//1、使用RSA中的GenerateKey方法生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	// 2、通过X509标准将得到的RAS私钥序列化为：ASN.1 的DER编码字符串
	privateStream := x509.MarshalPKCS1PrivateKey(privateKey)
	// 3、将私钥字符串设置到pem格式块中
	block1 := pem.Block{
		Type:  "private key",
		Bytes: privateStream,
	}
	// 4、通过pem将设置的数据进行编码，并写入磁盘文件
	fPrivate, err := os.Create("privateKey.pem")
	if err != nil {
		return err
	}
	defer fPrivate.Close()
	err = pem.Encode(fPrivate, &block1)
	if err != nil {
		return err
	}

	/*
		生成公钥
	*/
	publicKey := privateKey.PublicKey
	publicStream, err := x509.MarshalPKIXPublicKey(&publicKey)
	// publicStream:=x509.MarshalPKCS1PublicKey(&publicKey)
	block2 := pem.Block{
		Type:  "public key",
		Bytes: publicStream,
	}
	fPublic, err := os.Create("publicKey.pem")
	if err != nil {
		return err
	}
	defer fPublic.Close()
	pem.Encode(fPublic, &block2)
	return nil
}

// 对数据进行加密操作
func EncyptogRSA(src []byte, path string) (res []byte, err error) {
	// 1.获取秘钥（从本地磁盘读取）
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	fileInfo, _ := f.Stat()
	b := make([]byte, fileInfo.Size())
	f.Read(b)
	// 2、将得到的字符串解码
	block, _ := pem.Decode(b)
	println(base64.StdEncoding.EncodeToString(block.Bytes))
	bb, _ := base64.StdEncoding.DecodeString(`MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAzi1j6RjsQ/l0sSsR+SV5WVjl6QPRtd9X9flVwOS1pmRtpoEgvHSM6Q1tgdih/wXaFylgbNquULZ0Ld/8XPLHXY3nhW6fHmpd9O4oE4prGX7CKPLiQTatTy/S3vMbjR3lQP3mn+DAq4ygIWnnE4ZWCh4BvULuNp4ZPNUb8k2OX0wkidG+oAHmNqcRhvXWIFv3v80etgeOxjZXwLZjmMB+ZFWNaA4Ut8OCxXnxdNBt61EAxJsjWAWQ0aVLt8ZBp7yVolCz6thYPybNkjc3N/Oejt8pzpSgi5ZTBIBWRVJOhDC45okUCDXgXW6X5+UL7qFC54QGYcuKNcSgTIML+ZwGE/68G6mvIdUHG44nxbz9rfno02KNdFSxG0gTDNCqr/ifommR+nE/ggjvpJI6avrTldeyhzgUY/Q9/nuIdTyDYXWSbumE6iFaDagxQ8ay6EOCPE/bAdhsL99nT+v9xBG+FaHY9EOdi0TQsNmteu9L8l3+v6BDqn2Mt79tRob6nJu9zlbu8XFE5ASqRWUzLZ5Nxdr8eFHZGF3gHR/abVyNG8j4HAmFkkHlANg5nQnODukkyAWTJoYGcpqjSqOaJkyFk4mxmEf2SkXfI8reY8yx7niAbacy3DHhs6F9VF7M3GRMtymneQsqAihdQ2B8y9qT6KreEKmrCi/k9CmKLhrzFg8CAwEAAQ==`)
	// 使用X509将解码之后的数据 解析出来
	// x509.MarshalPKCS1PublicKey(block):解析之后无法用，所以采用以下方法：ParsePKIXPublicKey
	keyInit, err := x509.ParsePKIXPublicKey(bb) // 对应于生成秘钥的x509.MarshalPKIXPublicKey(&publicKey)
	// keyInit, err := x509.ParsePKCS1PublicKey(bb)
	if err != nil {
		return
	}
	// 4.使用公钥加密数据
	// pubKey := keyInit
	pubKey := keyInit.(*rsa.PublicKey)
	res, err = rsa.EncryptPKCS1v15(rand.Reader, pubKey, src)
	return
}

// 对数据进行解密操作
func DecrptogRSA(src []byte, path string) (res []byte, err error) {
	// 1.获取秘钥（从本地磁盘读取）
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	fileInfo, _ := f.Stat()
	b := make([]byte, fileInfo.Size())
	f.Read(b)
	block, _ := pem.Decode(b)                                 // 解码
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes) // 还原数据
	res, err = rsa.DecryptPKCS1v15(rand.Reader, privateKey, src)
	return
}

func addBase64Padding(value string) string {
	m := len(value) % 4
	if m != 0 {
		value += strings.Repeat("=", 4-m)
	}

	return value
}

func removeBase64Padding(value string) string {
	return strings.Replace(value, "=", "", -1)
}

func Pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func Unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}

	return src[:(length - unpadding)], nil
}

func encrypt(key, iv []byte, text string) (string, error) {
	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return "", err
	}

	msg := Pad([]byte(text))
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	// iv = ciphertext[:aes.BlockSize]
	// println(fmt.Sprintf("===========%+v", iv))
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	// println(fmt.Sprintf("===========%+v", iv))
	// io.ReadFull(rand.Reader, iv)
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(msg))
	finalMsg := hex.EncodeToString(ciphertext)
	return finalMsg, nil
}

func decrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	decodedMsg, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	println("=====", hex.EncodeToString(decodedMsg))
	if (len(decodedMsg) % aes.BlockSize) != 0 {
		return "", errors.New("blocksize must be multipe of decoded message length")
	}

	// iv := decodedMsg[:aes.BlockSize]
	iv := []byte("4qzB9DK6eFuSOMfB")
	msg := decodedMsg //[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(msg, msg)
	println("===++++++++++", len(msg))

	// unpadMsg, err := Unpad(msg)
	// if err != nil {
	// 	return "", err
	// }

	return string(msg), nil
}

// QueryDataRow 数据行
type QueryDataRow struct {
	Cells []config.VString `json:"cells,omitempty"`
}

// QueryData 数据集
type QueryData struct {
	Total    int32           `json:"total,omitempty"`
	CacheTag string          `json:"cache_tag,omitempty"`
	Rows     []*QueryDataRow `json:"rows,omitempty"`
	Columns  []string        `json:"columns,omitempty"`
}

func chtest(ch chan string) {
	time.Sleep(time.Second)
	ch <- gopsu.GetRandomString(10, true)
	time.Sleep(time.Second * 5)
	println("chtest done")
}

func main() {
	req, _ := http.NewRequest("GET", "http://127.0.0.1", nil)
	b, err := gopsu.DumpReqBody(req)
	if err != nil {
		println(err.Error())
		return
	}
	println(string(b))
}

var georep = strings.NewReplacer("(", "", ")", "", "POINT ", "", "POLYGON ", "", "LINESTRING ", "") // 经纬度字符串处理替换器

func text2Geo(s string) []*assetGeo {
	geostr := strings.Split(georep.Replace(s), ", ")
	gp := make([]*assetGeo, 0)
	for _, v := range geostr {
		vv := strings.Split(v, " ")
		gp = append(gp, &assetGeo{
			Lng: gopsu.String2Float64(vv[0]),
			Lat: gopsu.String2Float64(vv[1]),
		})
	}
	return gp
}

type assetGeo struct {
	Lng  float64 `json:"lng"`
	Lat  float64 `json:"lat"`
	Name string  `json:"aid,omitempty"`
}
