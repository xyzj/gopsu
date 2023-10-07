/*
Package gopsu ： 收集，保存的一些常用方法
*/
package gopsu

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/golang/snappy"
	"github.com/xyzj/gopsu/gocmd"
	json "github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/pathtool"
)

const (
	// OSNAME from runtime
	OSNAME = runtime.GOOS
	// OSARCH from runtime
	OSARCH = runtime.GOARCH
	// DateTimeFormat yyyy-mm-dd hh:MM:ss
	DateTimeFormat = "2006-01-02 15:04:05"
	// DateOnlyFormat yyyy-mm-dd hh:MM:ss
	DateOnlyFormat = "2006-01-02"
	// TimeOnlyFormat yyyy-mm-dd hh:MM:ss
	TimeOnlyFormat = "15:04:05"
	// LongTimeFormat 含日期的日志内容时间戳格式 2006/01/02 15:04:05.000
	LongTimeFormat = "2006-01-02 15:04:05.000"
	// ShortTimeFormat 无日期的日志内容时间戳格式 15:04:05.000
	ShortTimeFormat = "15:04:05.000"
	// FileTimeFormat 日志文件命名格式 060102
	FileTimeFormat = "060102" // 日志文件命名格式
)
const (
	// CryptoMD5 md5算法
	CryptoMD5 = iota
	// CryptoSHA256 sha256算法
	CryptoSHA256
	// CryptoSHA512 sha512算法
	CryptoSHA512
	// CryptoHMACSHA1 hmacsha1摘要算法
	CryptoHMACSHA1
	// CryptoHMACSHA256 hmacsha256摘要算法
	CryptoHMACSHA256
	// CryptoAES128CBC aes128cbc算法
	CryptoAES128CBC
	// CryptoAES128CFB aes128cfb算法
	CryptoAES128CFB
	// CryptoAES192CBC aes192cbc算法
	CryptoAES192CBC
	// CryptoAES192CFB aes192cfb算法
	CryptoAES192CFB
	// CryptoAES256CBC aes256cbc算法
	CryptoAES256CBC
	// CryptoAES256CFB aes256cfb算法
	CryptoAES256CFB
)

// SliceFlag 切片型参数，仅支持字符串格式
type SliceFlag []string

// String 返回参数
func (f *SliceFlag) String() string {
	return fmt.Sprintf("%v", []string(*f))
}

// Set 设置值
func (f *SliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// CryptoWorker 序列化或加密管理器
type CryptoWorker struct {
	cryptoType   byte
	cryptoHash   hash.Hash
	cryptoLocker sync.Mutex
	cryptoIV     []byte
	cryptoBlock  cipher.Block
}

var (
	// DefaultLogDir 默认日志文件夹
	DefaultLogDir = filepath.Join(pathtool.GetExecDir(), "..", "log")
	// DefaultCacheDir 默认缓存文件夹
	DefaultCacheDir = filepath.Join(pathtool.GetExecDir(), "..", "cache")
	// DefaultConfDir 默认配置文件夹
	DefaultConfDir = filepath.Join(pathtool.GetExecDir(), "..", "conf")
)

var (
	trimReplacer = strings.NewReplacer("\r", "", "\n", "", "\000", "", "\t", " ")
	httpClient   = &http.Client{
		Transport: &http.Transport{
			IdleConnTimeout:     time.Second * 10,
			MaxConnsPerHost:     777,
			MaxIdleConns:        1,
			MaxIdleConnsPerHost: 1,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
)

// DoRequestWithTimeout 发起请求
func DoRequestWithTimeout(req *http.Request, timeo time.Duration) (int, []byte, map[string]string, error) {
	// 处理头
	if req.Header.Get("Content-Type") == "" {
		switch req.Method {
		case "GET":
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		case "POST":
			req.Header.Set("Content-Type", "application/json")
		}
	}
	// 超时
	ctx, cancel := context.WithTimeout(context.Background(), timeo)
	defer cancel()
	// 请求
	start := time.Now()
	resp, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return 502, nil, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 502, nil, nil, err
	}
	end := time.Since(start).String()
	// 处理头
	h := make(map[string]string)
	h["resp_from"] = req.Host
	h["resp_duration"] = end
	for k := range resp.Header {
		h[k] = resp.Header.Get(k)
	}
	sc := resp.StatusCode
	return sc, b, h, nil
}

// GetNewCryptoWorker 获取新的序列化或加密管理器
// md5,sha256,sha512初始化后直接调用hash
// hmacsha1初始化后需调用SetSignKey设置签名key后调用hash
// aes加密算法初始化后需调用SetKey设置key和iv后调用Encrypt，Decrypt
func GetNewCryptoWorker(cryptoType byte) *CryptoWorker {
	h := &CryptoWorker{
		cryptoType: cryptoType,
	}
	switch cryptoType {
	case CryptoMD5:
		h.cryptoHash = md5.New()
	case CryptoSHA256:
		h.cryptoHash = sha256.New()
	case CryptoSHA512:
		h.cryptoHash = sha512.New()
	case CryptoHMACSHA1:
		h.cryptoHash = hmac.New(sha1.New, []byte{})
	case CryptoHMACSHA256:
		h.cryptoHash = hmac.New(sha256.New, []byte{})
	}
	return h
}

func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs5Unpadding(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}

// SetKey 设置aes-key,iv
func (h *CryptoWorker) SetKey(key, iv string) error {
	switch h.cryptoType {
	case CryptoHMACSHA1:
		h.cryptoHash = hmac.New(sha1.New, Bytes(key))
	case CryptoHMACSHA256:
		h.cryptoHash = hmac.New(sha256.New, Bytes(key))
	case CryptoAES128CBC:
		if len(key) < 16 || len(iv) < 16 {
			return fmt.Errorf("key length must be longer than 16, and the length of iv must be 16")
		}
		h.cryptoBlock, _ = aes.NewCipher(Bytes(key)[:16])
		h.cryptoIV = Bytes(iv)[:16]
	case CryptoAES192CBC:
		if len(key) < 24 || len(iv) < 16 {
			return fmt.Errorf("key length must be longer than 24, and the length of iv must be 16")
		}
		h.cryptoBlock, _ = aes.NewCipher(Bytes(key)[:24])
		h.cryptoIV = Bytes(iv)[:16]
	case CryptoAES256CBC:
		if len(key) < 32 || len(iv) < 16 {
			return fmt.Errorf("key length must be longer than 32, and the length of iv must be 16")
		}
		h.cryptoBlock, _ = aes.NewCipher(Bytes(key)[:32])
		h.cryptoIV = Bytes(iv)[:16]
	case CryptoAES128CFB:
		if len(key) < 16 || len(iv) < 16 {
			return fmt.Errorf("key length must be longer than 16, and the length of iv must be 16")
		}
		h.cryptoBlock, _ = aes.NewCipher(Bytes(key)[:16])
		h.cryptoIV = Bytes(iv)[:16]
	case CryptoAES192CFB:
		if len(key) < 24 || len(iv) < 16 {
			return fmt.Errorf("key length must be longer than 24, and the length of iv must be 16")
		}
		h.cryptoBlock, _ = aes.NewCipher(Bytes(key)[:24])
		h.cryptoIV = Bytes(iv)[:16]
	case CryptoAES256CFB:
		if len(key) < 32 || len(iv) < 16 {
			return fmt.Errorf("key length must be longer than 32, and the length of iv must be 16")
		}
		h.cryptoBlock, _ = aes.NewCipher(Bytes(key)[:32])
		h.cryptoIV = Bytes(iv)[:16]
	default:
		return fmt.Errorf("not yet supported")
	}
	return nil
}

// Encrypt 加密
func (h *CryptoWorker) Encrypt(s string) string {
	// h.cryptoLocker.Lock()
	// defer h.cryptoLocker.Unlock()
	if len(h.cryptoIV) == 0 {
		return ""
	}
	switch h.cryptoType {
	case CryptoAES128CBC, CryptoAES192CBC, CryptoAES256CBC:
		content := pkcs5Padding(Bytes(s), h.cryptoBlock.BlockSize())
		crypted := make([]byte, len(content))
		cipher.NewCBCEncrypter(h.cryptoBlock, h.cryptoIV).CryptBlocks(crypted, content)
		return base64.StdEncoding.EncodeToString(crypted)
	case CryptoAES128CFB, CryptoAES192CFB, CryptoAES256CFB:
		crypted := make([]byte, aes.BlockSize+len(s))
		cipher.NewCFBEncrypter(h.cryptoBlock, h.cryptoIV).XORKeyStream(crypted[aes.BlockSize:], Bytes(s))
		return base64.StdEncoding.EncodeToString(crypted)
	}
	return ""
}

// EncryptNoTail 加密，去掉base64尾巴的=符号
func (h *CryptoWorker) EncryptNoTail(s string) string {
	return strings.Replace(h.Encrypt(s), "=", "", -1)
}

// Decrypt 解密
func (h *CryptoWorker) Decrypt(s string) string {
	// h.cryptoLocker.Lock()
	// defer h.cryptoLocker.Unlock()
	defer func() { recover() }()
	if len(h.cryptoIV) == 0 {
		return ""
	}

	if x := 4 - len(s)%4; x != 4 {
		for i := 0; i < x; i++ {
			s += "="
		}
	}
	msg, _ := base64.StdEncoding.DecodeString(s)
	switch h.cryptoType {
	case CryptoAES128CBC, CryptoAES192CBC, CryptoAES256CBC:
		decrypted := make([]byte, len(msg))
		cipher.NewCBCDecrypter(h.cryptoBlock, h.cryptoIV).CryptBlocks(decrypted, msg)
		return String(pkcs5Unpadding(decrypted))
	case CryptoAES128CFB, CryptoAES192CFB, CryptoAES256CFB:
		msg = msg[aes.BlockSize:]
		cipher.NewCFBDecrypter(h.cryptoBlock, h.cryptoIV).XORKeyStream(msg, msg)
		return String(msg)
	}
	return ""
}

// Hash 计算序列
func (h *CryptoWorker) Hash(b []byte) string {
	h.cryptoLocker.Lock()
	defer h.cryptoLocker.Unlock()
	switch h.cryptoType {
	case CryptoMD5, CryptoSHA256, CryptoSHA512, CryptoHMACSHA1, CryptoHMACSHA256:
		h.cryptoHash.Reset()
		h.cryptoHash.Write(b)
		return fmt.Sprintf("%x", h.cryptoHash.Sum(nil))
	}
	return ""
}

// HashB64 返回base64编码格式
func (h *CryptoWorker) HashB64(b []byte) string {
	h.cryptoLocker.Lock()
	defer h.cryptoLocker.Unlock()
	switch h.cryptoType {
	case CryptoMD5, CryptoSHA256, CryptoSHA512, CryptoHMACSHA1, CryptoHMACSHA256:
		h.cryptoHash.Reset()
		h.cryptoHash.Write(b)
		return base64.StdEncoding.EncodeToString(h.cryptoHash.Sum(nil))
	}
	return ""
}

// GetMD5 生成32位md5字符串
func GetMD5(text string) string {
	ctx := md5.New()
	ctx.Write(Bytes(text))
	return hex.EncodeToString(ctx.Sum(nil))
}

// HashData 计算hash
func HashData(b []byte, cryptoType byte) string {
	switch cryptoType {
	case CryptoMD5:
		return fmt.Sprintf("%x", md5.Sum(b))
	case CryptoSHA256:
		return fmt.Sprintf("%x", sha256.Sum256(b))
	case CryptoSHA512:
		return fmt.Sprintf("%x", sha512.Sum512(b))
	}
	return ""
}

// ArchiveType 压缩编码类型
type ArchiveType byte

var (
	// ArchiveZlib zlib压缩/解压缩
	ArchiveZlib ArchiveType = 1
	// ArchiveGZip gzip压缩/解压缩
	ArchiveGZip ArchiveType = 2
	//ArchiveSnappy snappy压缩，解压缩
	ArchiveSnappy ArchiveType = 3
)

// ArchiveWorker 压缩管理器，避免重复New
type ArchiveWorker struct {
	archiveType      ArchiveType
	in               *bytes.Reader
	code             *bytes.Buffer
	decode           *bytes.Buffer
	gzipReader       *gzip.Reader
	gzipWriter       *gzip.Writer
	zlibReader       io.ReadCloser
	zlibWriter       *zlib.Writer
	snappyReader     *snappy.Reader
	snappyWriter     *snappy.Writer
	compressLocker   sync.Mutex
	uncompressLocker sync.Mutex
}

// GetNewArchiveWorker 获取新的压缩管理器
func GetNewArchiveWorker(archiveType ArchiveType) *ArchiveWorker {
	a := &ArchiveWorker{
		archiveType: archiveType,
		in:          bytes.NewReader(nil),
		code:        &bytes.Buffer{},
		decode:      &bytes.Buffer{},
	}
	switch archiveType {
	case ArchiveSnappy:
		a.snappyReader = snappy.NewReader(a.in)
		a.snappyWriter = snappy.NewBufferedWriter(a.code)
	case ArchiveGZip:
		a.gzipReader, _ = gzip.NewReader(a.in)
		a.gzipWriter = gzip.NewWriter(a.code)
	default:
		a.zlibReader, _ = zlib.NewReader(a.in)
		a.zlibWriter = zlib.NewWriter(a.code)
	}
	return a
}

// Compress 压缩
func (a *ArchiveWorker) Compress(src []byte) []byte {
	a.compressLocker.Lock()
	defer a.compressLocker.Unlock()
	a.code.Reset()
	switch a.archiveType {
	case ArchiveSnappy:
		a.snappyWriter.Reset(a.code)
		a.snappyWriter.Write(src)
		a.snappyWriter.Close()
	case ArchiveGZip:
		a.gzipWriter.Reset(a.code)
		a.gzipWriter.Write(src)
		a.gzipWriter.Close()
	default: // zlib
		a.zlibWriter.Reset(a.code)
		a.zlibWriter.Write(src)
		a.zlibWriter.Close()
	}
	return a.code.Bytes()
}

// Uncompress 解压缩
func (a *ArchiveWorker) Uncompress(src []byte) []byte {
	a.uncompressLocker.Lock()
	defer a.uncompressLocker.Unlock()
	a.decode.Reset()
	switch a.archiveType {
	case ArchiveSnappy:
		io.Copy(a.decode, snappy.NewReader(bytes.NewReader(src)))
	case ArchiveGZip:
		b := bytes.NewReader(src)
		r, _ := gzip.NewReader(b)
		io.Copy(a.decode, r)
	default: // zlib
		a.in.Reset(src)
		a.zlibReader, _ = zlib.NewReader(a.in)
		io.Copy(a.decode, a.zlibReader)
	}
	return a.decode.Bytes()
}

// CompressData 使用gzip，zlib压缩数据
func CompressData(src []byte, t ArchiveType) []byte {
	var in = &bytes.Buffer{}
	switch t {
	case ArchiveSnappy:
		w := snappy.NewBufferedWriter(in)
		w.Write(src)
		w.Close()
	case ArchiveGZip:
		w := gzip.NewWriter(in)
		w.Write(src)
		w.Close()
	default: // zlib
		w := zlib.NewWriter(in)
		w.Write(src)
		w.Close()
	}
	return in.Bytes()
}

// UncompressData 使用gzip，zlib解压缩数据
func UncompressData(src []byte, t ArchiveType, dstlen ...interface{}) []byte {
	var out = &bytes.Buffer{}
	switch t {
	case ArchiveSnappy:
		io.Copy(out, snappy.NewReader(bytes.NewReader(src)))
	case ArchiveGZip:
		b := bytes.NewReader(src)
		r, _ := gzip.NewReader(b)
		io.Copy(out, r)
	default: // zlib
		b := bytes.NewReader(src)
		r, _ := zlib.NewReader(b)
		io.Copy(out, r)
	}
	return out.Bytes()
}

// Base64URLDecode url解码
func Base64URLDecode(data string) ([]byte, error) {
	var missing = (4 - len(data)%4) % 4
	data += strings.Repeat("=", missing)
	res, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Base64UrlSafeEncode url safe 编码
func Base64UrlSafeEncode(source []byte) string {
	// Base64 Url Safe is the same as Base64 but does not contain '/' and '+' (replaced by '_' and '-') and trailing '=' are removed.
	bytearr := base64.StdEncoding.EncodeToString(source)
	// safeurl := strings.Replace(string(bytearr), "/", "_", -1)
	// safeurl = strings.Replace(safeurl, "+", "-", -1)
	// safeurl = strings.Replace(safeurl, "=", "", -1)
	return strings.NewReplacer("/", " ", "+", "-", "=", "").Replace(bytearr)
}

// StringSliceSort 字符串数组排序
type StringSliceSort struct {
	OneDimensional []string
	TwoDimensional [][]string
	Idx            int
	Order          string
}

func (arr *StringSliceSort) Len() int {
	if len(arr.OneDimensional) > 0 {
		return len(arr.OneDimensional)
	}
	return len(arr.TwoDimensional)
}

func (arr *StringSliceSort) Swap(i, j int) {
	if len(arr.OneDimensional) > 0 {
		arr.OneDimensional[i], arr.OneDimensional[j] = arr.OneDimensional[j], arr.OneDimensional[i]
	}
	arr.TwoDimensional[i], arr.TwoDimensional[j] = arr.TwoDimensional[j], arr.TwoDimensional[i]
}

func (arr *StringSliceSort) Less(i, j int) bool {
	if arr.Order == "desc" {
		if len(arr.OneDimensional) > 0 {
			return arr.OneDimensional[i] > arr.OneDimensional[j]
		}
		arr1 := arr.TwoDimensional[i]
		arr2 := arr.TwoDimensional[j]
		if arr.Idx > len(arr.TwoDimensional[0]) {
			arr.Idx = 0
		}
		return arr1[arr.Idx] > arr2[arr.Idx]
	}
	if len(arr.OneDimensional) > 0 {
		return arr.OneDimensional[i] < arr.OneDimensional[j]
	}
	arr1 := arr.TwoDimensional[i]
	arr2 := arr.TwoDimensional[j]
	if arr.Idx > len(arr.TwoDimensional[0]) {
		arr.Idx = 0
	}
	return arr1[arr.Idx] < arr2[arr.Idx]
}

// CacheMarshal 将数据进行序列化后压缩，可做数据缓存
func CacheMarshal(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return CompressData(b, ArchiveGZip), nil
}

// CacheUnmarshal 将压缩的数据反序列化，参数v必须专递结构地址
func CacheUnmarshal(b []byte, v interface{}) error {
	if err := json.Unmarshal(UncompressData(b, ArchiveGZip), v); err != nil {
		return err
	}
	return nil
}

// GetAddrFromString get addr from config string
//
// straddr: something like "1,2,3-6"
// return: []int64,something like []int64{1,2,3,4,5,6}
func GetAddrFromString(straddr string) ([]int64, error) {
	lst := strings.Split(strings.TrimSpace(straddr), ",")
	lstAddr := make([]int64, 0)
	for _, v := range lst {
		if strings.Contains(v, "-") {
			x := strings.Split(v, "-")
			x1, ex := strconv.ParseInt(strings.TrimSpace(x[0]), 10, 0)
			if ex != nil {
				return nil, ex
			}
			x2, ex := strconv.ParseInt(strings.TrimSpace(x[1]), 10, 0)
			if ex != nil {
				return nil, ex
			}
			for i := x1; i <= x2; i++ {
				lstAddr = append(lstAddr, i)
			}
		} else {
			y, ex := strconv.ParseInt(strings.TrimSpace(v), 10, 0)
			if ex != nil {
				return nil, ex
			}
			lstAddr = append(lstAddr, y)
		}
	}
	return lstAddr, nil
}

// CheckIP check if the ipstring is legal
//
//	 args:
//		ip: ipstring something like 127.0.0.1:10001
//	 return:
//		true/false
func CheckIP(ip string) bool {
	regip := `^(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)$`
	regipwithport := `^(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d):\d{1,5}$`
	if strings.Contains(ip, ":") {
		a, ex := regexp.Match(regipwithport, Bytes(ip))
		if ex != nil {
			return false
		}
		s := strings.Split(ip, ":")[1]
		if p, err := strconv.Atoi(s); err != nil || p > 65535 {
			return false
		}
		return a
	}
	a, ex := regexp.Match(regip, Bytes(ip))
	if ex != nil {
		return false
	}
	return a
}

// MakeRuntimeDirs make conf,log,cache dirs
// Args：
// rootpath： 输入路径
// return：
// conf，log，cache三个文件夹的完整路径
func MakeRuntimeDirs(rootpath string) (string, string, string) {
	var basepath string
	switch rootpath {
	case ".":
		basepath = pathtool.GetExecDir()
	case "..":
		basepath = pathtool.JoinPathFromHere("..")
	default:
		basepath = rootpath
	}
	os.MkdirAll(filepath.Join(basepath, "conf"), 0775)
	os.MkdirAll(filepath.Join(basepath, "log"), 0775)
	os.MkdirAll(filepath.Join(basepath, "cache"), 0775)
	return filepath.Join(basepath, "conf"), filepath.Join(basepath, "log"), filepath.Join(basepath, "cache")
}

// CheckLrc check lrc data
func CheckLrc(d []byte) bool {
	rowdata := d[:len(d)-1]
	lrcdata := d[len(d)-1]

	c := CountLrc(&rowdata)
	return c == lrcdata
}

// CountLrc count lrc data
func CountLrc(data *[]byte) byte {
	a := byte(0)
	for _, v := range *data {
		a ^= v
	}
	return a
}

// CheckCrc16VBBigOrder check crc16 data，use big order
func CheckCrc16VBBigOrder(d []byte) bool {
	rowdata := d[:len(d)-2]
	crcdata := d[len(d)-2:]

	c := CountCrc16VB(&rowdata)
	if c[1] == crcdata[0] && c[0] == crcdata[1] {
		return true
	}
	return false
}

// CheckCrc16VB check crc16 data
func CheckCrc16VB(d []byte) bool {
	rowdata := d[:len(d)-2]
	crcdata := d[len(d)-2:]

	c := CountCrc16VB(&rowdata)
	if c[0] == crcdata[0] && c[1] == crcdata[1] {
		return true
	}
	return false
}

// CountCrc16VB count crc16 as vb6 do
func CountCrc16VB(data *[]byte) []byte {
	crc16lo := byte(0xFF)
	crc16hi := byte(0xFF)
	cl := byte(0x01)
	ch := byte(0xa0)
	for _, v := range *data {
		crc16lo ^= v
		for i := 0; i < 8; i++ {
			savehi := crc16hi
			savelo := crc16lo
			crc16hi /= 2
			crc16lo /= 2
			if (savehi & 0x01) == 0x01 {
				crc16lo ^= 0x80
			}
			if (savelo & 0x01) == 0x01 {
				crc16hi ^= ch
				crc16lo ^= cl
			}
		}
	}
	return []byte{crc16lo, crc16hi}
}

// SplitDateTime SplitDateTime
func SplitDateTime(sdt int64) (y, m, d, h, mm, s, wd byte) {
	if sdt == 0 {
		sdt = time.Now().Unix()
	}
	if sdt > 621356256000000000 {
		sdt = SwitchStamp(sdt)
	}
	tm := time.Unix(sdt, 0)
	stm := tm.Format("2006-01-02 15:04:05 Mon")
	dt := strings.Split(stm, " ")
	dd := strings.Split(dt[0], "-")
	tt := strings.Split(dt[1], ":")
	return byte(String2Int32(dd[0], 10) - 2000),
		String2Byte(dd[1], 10),
		String2Byte(dd[2], 10),
		String2Byte(tt[0], 10),
		String2Byte(tt[1], 10),
		String2Byte(tt[2], 10),
		byte(tm.Weekday())
}

// ReverseString ReverseString
func ReverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

// CodeString 编码字符串
func CodeString(s string) string {
	x := byte(rand.Int31n(126) + 1)
	l := len(s)
	salt := GetRandomASCII(int64(l))
	var y, z bytes.Buffer
	for _, v := range Bytes(s) {
		y.WriteByte(v + x)
	}
	zz := y.Bytes()
	var c1, c2 int
	z.WriteByte(x)
	for i := 1; i < 2*l; i++ {
		if i%2 == 0 {
			z.WriteByte(salt[c1])
			c1++
		} else {
			z.WriteByte(zz[c2])
			c2++
		}
	}
	a := base64.StdEncoding.EncodeToString(z.Bytes())
	a = ReverseString(a)
	a = SwapCase(a)
	a = strings.Replace(a, "=", "", -1)
	return a
}

// DecodeString 解码混淆字符串，兼容python算法
func DecodeString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}
	s = ReverseString(SwapCase(s))
	if x := 4 - len(s)%4; x != 4 {
		for i := 0; i < x; i++ {
			s += "="
		}
	}
	if y, ex := base64.StdEncoding.DecodeString(s); ex == nil {
		var ns bytes.Buffer
		x := y[0]
		for k, v := range y {
			if k%2 != 0 {
				ns.WriteByte(v - x)
			}
		}
		return ns.String()
	}
	return ""
}

// DecodeStringOld 解码混淆字符串，兼容python算法
func DecodeStringOld(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}
	s = SwapCase(s)
	var ns bytes.Buffer
	ns.Write([]byte{120, 156})
	if x := 4 - len(s)%4; x != 4 {
		for i := 0; i < x; i++ {
			s += "="
		}
	}
	if y, ex := base64.StdEncoding.DecodeString(s); ex == nil {
		x := String2Byte(string(y[0])+string(y[1]), 0)
		z := y[2:]
		for i := len(z) - 1; i >= 0; i-- {
			if z[i] >= x {
				ns.WriteByte(z[i] - x)
			} else {
				ns.WriteByte(byte(int(z[i]) + 256 - int(x)))
			}
		}
		return ReverseString(String(DoZlibUnCompress(ns.Bytes())))
	}
	return ""
}

// DoZlibUnCompress zlib uncompress
func DoZlibUnCompress(src []byte) []byte {
	b := bytes.NewReader(src)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	io.Copy(&out, r)
	return out.Bytes()
}

// DoZlibCompress zlib compress
func DoZlibCompress(src []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

// SwapCase swap char case
func SwapCase(s string) string {
	var ns bytes.Buffer
	for _, v := range s {
		if v >= 65 && v <= 90 {
			ns.WriteString(string(v + 32))
		} else if v >= 97 && v <= 122 {
			ns.WriteString(string(v - 32))
		} else {
			ns.WriteString(string(v))
		}
	}
	return ns.String()
}

// VersionInfo show something
//
// name: program name
// ver: program version
// gover: golang version
// buildDate: build datetime
// buildOS: platform info
// auth: auth name
func VersionInfo(name, ver, gover, buildDate, buildOS, auth string) string {
	return gocmd.PrintVersion(&gocmd.VersionInfo{
		Name:      name,
		Version:   ver,
		GoVersion: gover,
		BuildDate: buildDate,
		BuildOS:   buildOS,
		CodeBy:    auth,
	})
}

// WriteVersionInfo write version info to .ver file
//
//	 args:
//		p: program name
//		v: program version
//		gv: golang version
//		bd: build datetime
//		pl: platform info
//		a: auth name
func WriteVersionInfo(p, v, gv, bd, pl, a string) {
	fn, _ := os.Executable()
	f, _ := os.OpenFile(fmt.Sprintf("%s.ver", fn), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0444)
	defer f.Close()
	f.WriteString(fmt.Sprintf("\n%s\r\nVersion:\t%s\r\nGo version:\t%s\r\nBuild date:\t%s\r\nBuild OS:\t%s\r\nCode by:\t%s\r\nStart with:\t%s", p, v, gv, pl, bd, a, strings.Join(os.Args[1:], " ")))
}

// CalculateSecurityCode calculate security code
//
//	 args:
//		t: calculate type "h"-按小时计算，当分钟数在偏移值范围内时，同时计算前后一小时的值，"m"-按分钟计算，同时计算前后偏移量范围内的值
//		salt: 拼接用字符串
//		offset: 偏移值，范围0～59
//	 return:
//		32位小写md5码切片
func CalculateSecurityCode(t, salt string, offset int) []string {
	var sc []string
	if offset < 0 {
		offset = 0
	}
	if offset > 59 {
		offset = 59
	}
	tt := time.Now()
	mm := tt.Minute()
	switch t {
	case "h":
		sc = make([]string, 0, 3)
		sc = append(sc, GetMD5(tt.Format("2006010215")+salt))
		if mm < offset || 60-mm < offset {
			sc = append(sc, GetMD5(tt.Add(-1*time.Hour).Format("2006010215")+salt))
			sc = append(sc, GetMD5(tt.Add(time.Hour).Format("2006010215")+salt))
		}
	case "m":
		sc = make([]string, 0, offset*2)
		if offset > 0 {
			tts := tt.Add(time.Duration(-1*(offset)) * time.Minute)
			for i := 0; i < offset*2+1; i++ {
				sc = append(sc, GetMD5(tts.Add(time.Duration(i)*time.Minute).Format("200601021504")+salt))
			}
		} else {
			sc = append(sc, GetMD5(tt.Format("200601021504")+salt))
		}
	}
	return sc
}

// GetRandomASCII 获取随机ascII码字符串
func GetRandomASCII(l int64) []byte {
	var rs bytes.Buffer
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := int64(0); i < l; i++ {
		rs.WriteByte(byte(rand.Int31n(255) + 1))
	}
	return rs.Bytes()
}

// GetRandomString 生成随机字符串
func GetRandomString(l int64, letteronly ...bool) string {
	str := "!#%&()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}"
	if len(letteronly) > 0 && letteronly[0] {
		str = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}
	bb := Bytes(str)
	var rs strings.Builder
	// var rs bytes.Buffer
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := int64(0); i < l; i++ {
		rs.WriteByte(bb[rand.Intn(len(bb))])
		// rs.WriteByte(bb[rand.Intn(len(bb))])
	}
	return rs.String()
}

// CheckSQLInject 检查sql语句是否包含注入攻击
func CheckSQLInject(s string) bool {
	str := `(?:')|(?:--)|(/\\*(?:.|[\\n\\r])*?\\*/)|(\b(select|update|and|or|delete|insert|trancate|char|chr|into|substr|ascii|declare|exec|count|master|into|drop|execute)\b)`
	re, err := regexp.Compile(str)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

// Bytes2Int64 字节数组转换为int64，bigOrder==true,高位在前
// func Bytes2Int64(b []byte, bigOrder bool) int64 {
// 	var l = len(b)
// 	switch l {
// 	case 1:
// 		var tmp int8
// 		bytesBuffer := bytes.NewBuffer(b)
// 		if bigOrder {
// 			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
// 		} else {
// 			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
// 		}
// 		return int64(tmp)
// 	case 2:
// 		var tmp int16
// 		bytesBuffer := bytes.NewBuffer(b)
// 		if bigOrder {
// 			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
// 		} else {
// 			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
// 		}
// 		return int64(tmp)
// 	case 3, 4:
// 		var tmp int32
// 		bytesBuffer := bytes.NewBuffer(b)
// 		if bigOrder {
// 			if l == 3 {
// 				b = append([]byte{0}, b...)
// 			}
// 			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
// 		} else {
// 			if l == 3 {
// 				b = append(b, 0)
// 			}
// 			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
// 		}
// 		return int64(tmp)
// 	case 5, 6, 7, 8:
// 		var tmp int64
// 		bytesBuffer := bytes.NewBuffer(b)
// 		if bigOrder {
// 			if l < 8 {
// 				bb := make([]byte, 8-l)
// 				b = append(bb, b...)
// 			}
// 			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
// 		} else {
// 			if l < 8 {
// 				bb := make([]byte, 8-l)
// 				b = append(b, bb...)
// 			}
// 			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
// 		}
// 		return int64(tmp)
// 	}
// 	return 0
// }

// // Bytes2Uint64 字节数组转换为uint64，bigOrder==true,高位在前
// func Bytes2Uint64(b []byte, bigOrder bool) uint64 {
// 	var l int
// 	if len(b) > 8 {
// 		l = 0
// 	} else {
// 		l = 8 - len(b)
// 	}
// 	var bb = make([]byte, l)
// 	if bigOrder {
// 		bb = append(bb, b...)
// 		b = bb
// 	} else {
// 		b = append(b, bb...)
// 	}
// 	if bigOrder {
// 		return binary.BigEndian.Uint64(b)
// 	} else {
// 		return binary.LittleEndian.Uint64(b)
// 	}
// }

// GetServerTLSConfig 获取https配置
//
//	certfile: 服务端证书
//	keyfile: 服务端key
//	clientca: 双向验证时客户端根证书
func GetServerTLSConfig(certfile, keyfile, clientca string) (*tls.Config, error) {
	tc := &tls.Config{}
	cliCrt, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	tc.Certificates = []tls.Certificate{cliCrt}
	caCrt, err := os.ReadFile(clientca)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if pool.AppendCertsFromPEM(caCrt) {
		tc.ClientCAs = pool
		tc.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return tc, nil
}

// GetClientTLSConfig 获取https配置
//
//	certfile: 双向验证时客户端证书
//	keyfile: 双向验证时客户端key
//	rootca: 服务端根证书
func GetClientTLSConfig(certfile, keyfile, rootca string) (*tls.Config, error) {
	tc := &tls.Config{
		InsecureSkipVerify: true,
	}
	var err error
	caCrt, err := os.ReadFile(rootca)
	if err == nil {
		pool := x509.NewCertPool()
		if pool.AppendCertsFromPEM(caCrt) {
			tc.RootCAs = pool
		}
	}
	cliCrt, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return tc, nil
	}
	tc.Certificates = []tls.Certificate{cliCrt}
	return tc, nil
}

// TrimString 去除字符串末尾的空格，\r\n
func TrimString(s string) string {
	return trimReplacer.Replace(strings.TrimSpace(s))
}

// ZIPFiles 压缩多个文件
func ZIPFiles(dstName string, srcFiles []string, newDir string) error {
	newZipFile, err := os.Create(dstName)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()
	for _, v := range srcFiles {
		zipfile, err := os.Open(v)
		if err != nil {
			return err
		}
		defer zipfile.Close()
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Method = zip.Deflate
		switch newDir {
		case "":
			header.Name = filepath.Base(v)
		case "/":
			header.Name = v
		default:
			header.Name = filepath.Join(newDir, filepath.Base(v))
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err = io.Copy(writer, zipfile); err != nil {
			return err
		}
	}
	return nil
}

// UnZIPFile 解压缩文件
func UnZIPFile(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	if target != "" {
		if err := os.MkdirAll(target, 0775); err != nil {
			return err
		}
	} else {
		target = "."
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, 0775)
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}
	return nil
}

// ZIPFile 压缩文件
func ZIPFile(d, s string, delold bool) error {
	zfile := filepath.Join(d, s+".zip")
	ofile := filepath.Join(d, s)

	newZipFile, err := os.Create(zfile)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	zipfile, err := os.Open(ofile)
	if err != nil {
		return err
	}
	defer zipfile.Close()
	info, err := zipfile.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err = io.Copy(writer, zipfile); err != nil {
		return err
	}
	if delold {
		go func(s string) {
			time.Sleep(time.Second * 10)
			os.Remove(s)
		}(filepath.Join(d, s))
	}
	return nil
}

// CountRCMru 计算电表校验码
func CountRCMru(d []byte) byte {
	var a int
	for _, v := range d {
		a += int(v)
	}
	return byte(a % 256)
}

// CheckRCMru 校验电表数据
func CheckRCMru(d []byte) bool {
	return d[len(d)-2] == CountRCMru(d[:len(d)-2])
}

// CopyFile 复制文件
func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

// IsExist file is exist or not
func IsExist(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}

// JoinPathFromHere 从程序执行目录开始拼接路径
func JoinPathFromHere(path ...string) string {
	s := []string{pathtool.GetExecDir()}
	s = append(s, path...)
	sp := filepath.Join(s...)
	p, err := filepath.Abs(sp)
	if err != nil {
		return sp
	}
	return p
}

// SlicesUnion 求并集
func SlicesUnion(slice1, slice2 []string) []string {
	m := make(map[string]int)
	for _, v := range slice1 {
		if v == "" {
			continue
		}
		m[v]++
	}

	for _, v := range slice2 {
		if v == "" {
			continue
		}
		if _, ok := m[v]; !ok {
			slice1 = append(slice1, v)
		}
	}
	return slice1
}

// SlicesIntersect 求交集
func SlicesIntersect(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	for _, v := range slice1 {
		if v == "" {
			continue
		}
		m[v]++
	}

	for _, v := range slice2 {
		if v == "" {
			continue
		}
		if _, ok := m[v]; ok {
			nn = append(nn, v)
		}
	}
	return nn
}

// SlicesDifference 求差集 slice1-并集
func SlicesDifference(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	inter := SlicesIntersect(slice1, slice2)
	for _, v := range inter {
		if v == "" {
			continue
		}
		m[v]++
	}
	union := SlicesUnion(slice1, slice2)
	for _, v := range union {
		if v == "" {
			continue
		}
		if _, ok := m[v]; !ok {
			nn = append(nn, v)
		}
	}
	return nn
}

// CalcCRC32String 计算crc32，返回16进制字符串
func CalcCRC32String(b []byte) string {
	return strconv.FormatUint(uint64(crc32.ChecksumIEEE(b)), 16)
}

// CalcCRC32 计算crc32，返回[]byte
func CalcCRC32(b []byte, bigorder bool) []byte {
	return HexString2Bytes(strconv.FormatUint(uint64(crc32.ChecksumIEEE(b)), 16), bigorder)
}

// GetTCPPort 获取随机可用端口
func GetTCPPort() (int, error) {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", "0.0.0.0"))
	if err != nil {
		return 0, err
	}
	var listener *net.TCPListener
	var found = false
	for i := 0; i < 100; i++ {
		listener, err = net.ListenTCP("tcp", address)
		if err != nil {
			continue
		}
		found = true
	}
	defer listener.Close()
	if !found {
		return 0, fmt.Errorf("could not find a useful port")
	}
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// LastSlice 返回切片的最后一个元素
func LastSlice(s, sep string) string {
	ss := strings.Split(s, sep)
	if len(ss) > 0 {
		return ss[len(ss)-1]
	}
	return s
}

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
		}{s, len(s)},
	))
	// x := (*[2]uintptr)(unsafe.Pointer(&s))
	// h := [3]uintptr{x[0], x[1], x[1]}
	// return *(*[]byte)(unsafe.Pointer(&h))
}

// FormatFileSize 字节的单位转换
func FormatFileSize(fileSize int64) (size string) {
	if fileSize < 1024 {
		//return strconv.FormatInt(fileSize, 10) + "B"
		return fmt.Sprintf("%d B", fileSize/1)
	} else if fileSize < (1024 * 1024) {
		return fmt.Sprintf("%d KB", fileSize/1024)
	} else if fileSize < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%d MB", fileSize/(1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%d GB", fileSize/(1024*1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%d TB", fileSize/(1024*1024*1024*1024))
	} else { //if fileSize < (1024 * 1024 * 1024 * 1024 * 1024 * 1024)
		return fmt.Sprintf("%d EB", fileSize/(1024*1024*1024*1024*1024))
	}
}

// EraseSyncMap 清空sync.map
func EraseSyncMap(m *sync.Map) {
	m.Range(func(key interface{}, value interface{}) bool {
		m.Delete(key)
		return true
	})
}

// PB2Json pb2格式转换为json []byte格式
func PB2Json(pb interface{}) []byte {
	jsonBytes, err := json.Marshal(pb)
	if err != nil {
		return nil
	}
	return jsonBytes
}

// PB2String pb2格式转换为json 字符串格式
func PB2String(pb interface{}) string {
	b, err := json.Marshal(pb)
	if err != nil {
		return ""
	}
	return String(b)
}

// JSON2PB json字符串转pb2格式
func JSON2PB(js string, pb interface{}) error {
	err := json.Unmarshal(Bytes(js), &pb)
	return err
}
