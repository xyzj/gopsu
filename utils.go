package gopsu

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"container/list"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/golang/snappy"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
)

const (
	// OSNAME from runtime
	OSNAME = runtime.GOOS
	// OSARCH from runtime
	OSARCH = runtime.GOARCH
	// LongTimeFormat 含日期的日志内容时间戳格式 2006/01/02 15:04:05.000
	LongTimeFormat = "2006/01/02 15:04:05.000"
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

// CryptoWorker 序列化或加密管理器
type CryptoWorker struct {
	cryptoType         byte
	cryptoHash         hash.Hash
	cryptoLocker       sync.Mutex
	cryptoKey          []byte
	cryptoIV           []byte
	cryptoBlock        cipher.Block
	cryptoCBCEncrypter cipher.BlockMode
	cryptoCBCDecrypter cipher.BlockMode
	cryptoCFBEncrypter cipher.Stream
	cryptoCFBDecrypter cipher.Stream
}

var (
	// DefaultLogDir 默认日志文件夹
	DefaultLogDir = filepath.Join(GetExecDir(), "..", "log")
	// DefaultCacheDir 默认缓存文件夹
	DefaultCacheDir = filepath.Join(GetExecDir(), "..", "cache")
	// DefaultConfDir 默认配置文件夹
	DefaultConfDir = filepath.Join(GetExecDir(), "..", "conf")
)

var json = jsoniter.Config{}.Froze()

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GetNewCryptoWorker 获取新的序列化或加密管理器
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
	case CryptoAES128CBC:
		if len(key) < 16 || len(iv) < 16 {
			return fmt.Errorf("key and iv must be longer than 16")
		}
		h.cryptoBlock, _ = aes.NewCipher([]byte(key)[:16])
		h.cryptoIV = []byte(iv)[:16]
	case CryptoAES192CBC:
		if len(key) < 24 || len(iv) < 24 {
			return fmt.Errorf("key and iv must be longer than 24")
		}
		h.cryptoBlock, _ = aes.NewCipher([]byte(key)[:24])
		h.cryptoIV = []byte(iv)[:24]
	case CryptoAES256CBC:
		if len(key) < 32 || len(iv) < 32 {
			return fmt.Errorf("key and iv must be longer than 32")
		}
		h.cryptoBlock, _ = aes.NewCipher([]byte(key)[:32])
		h.cryptoIV = []byte(iv)[:32]
	case CryptoAES128CFB:
		if len(key) < 16 || len(iv) < 16 {
			return fmt.Errorf("key and iv must be longer than 16")
		}
		h.cryptoBlock, _ = aes.NewCipher([]byte(key)[:16])
		h.cryptoIV = []byte(iv)[:16]
	case CryptoAES192CFB:
		if len(key) < 24 || len(iv) < 24 {
			return fmt.Errorf("key and iv must be longer than 24")
		}
		h.cryptoBlock, _ = aes.NewCipher([]byte(key)[:24])
		h.cryptoIV = []byte(iv)[:24]
	case CryptoAES256CFB:
		if len(key) < 32 || len(iv) < 32 {
			return fmt.Errorf("key and iv must be longer than 32")
		}
		h.cryptoBlock, _ = aes.NewCipher([]byte(key)[:32])
		h.cryptoIV = []byte(iv)[:32]
	default:
		return fmt.Errorf("Not yet supported")
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
		content := pkcs5Padding([]byte(s), h.cryptoBlock.BlockSize())
		crypted := make([]byte, len(content))
		cipher.NewCBCEncrypter(h.cryptoBlock, h.cryptoIV).CryptBlocks(crypted, content)
		return base64.StdEncoding.EncodeToString(crypted)
	case CryptoAES128CFB, CryptoAES192CFB, CryptoAES256CFB:
		crypted := make([]byte, aes.BlockSize+len(s))
		cipher.NewCFBEncrypter(h.cryptoBlock, h.cryptoIV).XORKeyStream(crypted[aes.BlockSize:], []byte(s))
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
		return string(pkcs5Unpadding(decrypted))
	case CryptoAES128CFB, CryptoAES192CFB, CryptoAES256CFB:
		msg = msg[aes.BlockSize:]
		cipher.NewCFBDecrypter(h.cryptoBlock, h.cryptoIV).XORKeyStream(msg, msg)
		return string(msg)
	}
	return ""
}

// Hash 计算序列
func (h *CryptoWorker) Hash(b []byte) string {
	h.cryptoLocker.Lock()
	defer h.cryptoLocker.Unlock()
	switch h.cryptoType {
	case CryptoMD5, CryptoSHA256, CryptoSHA512:
		h.cryptoHash.Reset()
		h.cryptoHash.Write(b)
		return fmt.Sprintf("%x", h.cryptoHash.Sum(nil))
	}
	return ""
}

// GetMD5 生成32位md5字符串
func GetMD5(text string) string {
	ctx := md5.New()
	ctx.Write([]byte(text))
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

const (
	// ArchiveZlib zlib压缩/解压缩
	ArchiveZlib = iota
	// ArchiveGZip gzip压缩/解压缩
	ArchiveGZip
	// ArchiveSnappy snappy压缩/解压缩
	ArchiveSnappy
)

// ArchiveWorker 压缩管理器，避免重复New
type ArchiveWorker struct {
	archiveType      byte
	in               *bytes.Reader
	code             bytes.Buffer
	decode           bytes.Buffer
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
func GetNewArchiveWorker(archiveType byte) *ArchiveWorker {
	a := &ArchiveWorker{
		archiveType: archiveType,
		in:          bytes.NewReader(nil),
	}
	switch archiveType {
	case ArchiveGZip:
		a.gzipReader, _ = gzip.NewReader(a.in)
		a.gzipWriter = gzip.NewWriter(&a.code)
	case ArchiveSnappy:
		a.snappyReader = snappy.NewReader(a.in)
		a.snappyWriter = snappy.NewWriter(&a.code)
	default:
		a.zlibReader, _ = zlib.NewReader(a.in)
		a.zlibWriter = zlib.NewWriter(&a.code)
	}
	return a
}

// Compress 压缩
func (a *ArchiveWorker) Compress(src []byte) []byte {
	a.compressLocker.Lock()
	defer a.compressLocker.Unlock()
	a.code.Reset()
	switch a.archiveType {
	case ArchiveGZip:
		a.gzipWriter.Reset(&a.code)
		a.gzipWriter.Write(src)
		a.gzipWriter.Close()
	case ArchiveSnappy:
		a.code.Write(snappy.Encode(nil, src))
	default: // zlib
		a.zlibWriter.Reset(&a.code)
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
	case ArchiveGZip:
		b := bytes.NewReader(src)
		r, _ := gzip.NewReader(b)
		io.Copy(&a.decode, r)
	case ArchiveSnappy:
		b, err := snappy.Decode(nil, src)
		if err == nil {
			a.decode.Write(b)
		}
	default: // zlib
		a.in.Reset(src)
		a.zlibReader, _ = zlib.NewReader(a.in)
		io.Copy(&a.decode, a.zlibReader)
	}
	return a.decode.Bytes()
}

// CompressData 使用gzip，zlib压缩数据
func CompressData(src []byte, t byte) []byte {
	var in bytes.Buffer
	switch t {
	case ArchiveGZip:
		w := gzip.NewWriter(&in)
		w.Write(src)
		w.Close()
	case ArchiveSnappy:
		in.Write(snappy.Encode(nil, src))
	default: // zlib
		w := zlib.NewWriter(&in)
		w.Write(src)
		w.Close()
	}
	return in.Bytes()
}

// UncompressData 使用gzip，zlib解压缩数据
func UncompressData(src []byte, t byte, dstlen ...interface{}) []byte {
	var out bytes.Buffer
	switch t {
	case ArchiveGZip:
		b := bytes.NewReader(src)
		r, _ := gzip.NewReader(b)
		io.Copy(&out, r)
	case ArchiveSnappy:
		b, err := snappy.Decode(nil, src)
		if err == nil {
			out.Write(b)
		}
	default: // zlib
		b := bytes.NewReader(src)
		r, _ := zlib.NewReader(b)
		io.Copy(&out, r)
	}
	return out.Bytes()
}

// GetUUID1 GetUUID1
func GetUUID1() string {
	uid, _ := uuid.NewUUID()
	return uid.String()
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

// Queue queue for go
type Queue struct {
	q      *list.List
	locker *sync.Mutex
}

// NewQueue get a new queue
func NewQueue() *Queue {
	mq := &Queue{
		q:      list.New(),
		locker: &sync.Mutex{},
	}
	return mq
}

// Clear clear queue list
func (mq *Queue) Clear() {
	mq.q.Init()
}

// Put put data to the end of the queue
func (mq *Queue) Put(value interface{}) {
	mq.locker.Lock()
	defer mq.locker.Unlock()
	mq.q.PushBack(value)
}

// PutFront put data to the first of the queue
func (mq *Queue) PutFront(value interface{}) {
	mq.locker.Lock()
	defer mq.locker.Unlock()
	mq.q.PushFront(value)
}

// Get get data from front
func (mq *Queue) Get() interface{} {
	if mq.q.Len() == 0 {
		return nil
	}
	mq.locker.Lock()
	defer mq.locker.Unlock()
	e := mq.q.Front()
	if e != nil {
		mq.q.Remove(e)
		return e.Value
	}
	return nil
}

// GetNoDel get data from front
func (mq *Queue) GetNoDel() interface{} {
	if mq.q.Len() == 0 {
		return nil
	}
	mq.locker.Lock()
	defer mq.locker.Unlock()
	e := mq.q.Front()
	return e.Value
}

// Len get queue len
func (mq *Queue) Len() int64 {
	return int64(mq.q.Len())
}

// Empty check if empty
func (mq *Queue) Empty() bool {
	return mq.q.Len() == 0
}

// GetAddrFromString get addr from config string
//  args:
//	straddr: something like "1,2,3-6"
//  return:
//	[]int64,something like []int64{1,2,3,4,5,6}
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
//  args:
//	ip: ipstring something like 127.0.0.1:10001
//  return:
//	true/false
func CheckIP(ip string) bool {
	regip := `^(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)$`
	regipwithport := `^(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d):\d{1,5}$`
	if strings.Contains(ip, ":") {
		a, ex := regexp.Match(regipwithport, []byte(ip))
		if ex != nil {
			return false
		}
		s := strings.Split(ip, ":")[1]
		if p, err := strconv.Atoi(s); err != nil || p > 65535 {
			return false
		}
		return a
	}
	a, ex := regexp.Match(regip, []byte(ip))
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
	if strings.Compare(rootpath, ".") == 0 {
		basepath = GetExecDir()
	} else {
		basepath = rootpath
	}
	os.MkdirAll(filepath.Join(basepath, "..", "conf"), 0775)
	os.MkdirAll(filepath.Join(basepath, "..", "log"), 0775)
	os.MkdirAll(filepath.Join(basepath, "..", "cache"), 0775)
	return filepath.Join(basepath, "..", "conf"), filepath.Join(basepath, "..", "log"), filepath.Join(basepath, "..", "cache")
}

// String2Bytes convert hex-string to []byte
//  args:
// 	data: 输入字符串
// 	sep： 用于分割字符串的分割字符
//  return:
// 	字节切片
func String2Bytes(data string, sep string) []byte {
	var z []byte
	a := strings.Split(data, sep)
	z = make([]byte, len(a))
	for k, v := range a {
		b, _ := strconv.ParseUint(v, 16, 8)
		z[k] = byte(b)
	}
	return z
}

// Bytes2String convert []byte to hex-string
//  args:
// 	data: 输入字节切片
// 	sep： 用于分割字符串的分割字符
//  return:
// 	字符串
func Bytes2String(data []byte, sep string) string {
	a := make([]string, len(data))
	for k, v := range data {
		a[k] = fmt.Sprintf("%02x", v)
	}
	return strings.Join(a, sep)
}

// String2Int convert string 2 int
//  args:
// 	s: 输入字符串
// 	t: 返回数值进制
//  Return：
// 	int
func String2Int(s string, t int) int {
	x, _ := strconv.ParseInt(s, t, 0)
	return int(x)
}

// String2Int8 convert string 2 int8
//  args:
// 	s: 输入字符串
// 	t: 返回数值进制
//  Return：
// 	int8
func String2Int8(s string, t int) byte {
	x, _ := strconv.ParseInt(s, t, 0)
	return byte(x)
}

// String2Int32 convert string 2 int32
//  args:
// 	s: 输入字符串
// 	t: 返回数值进制
//  Return：
// 	int32
func String2Int32(s string, t int) int32 {
	x, _ := strconv.ParseInt(s, t, 0)
	return int32(x)
}

// String2Int64 convert string 2 int64
//  args:
// 	s: 输入字符串
// 	t: 返回数值进制
//  Return：
// 	int64
func String2Int64(s string, t int) int64 {
	x, _ := strconv.ParseInt(s, t, 0)
	return x
}

// String2Float64 convert string 2 float64
func String2Float64(s string) float64 {
	x, _ := strconv.ParseFloat(s, 0)
	return x
}

//StringSlice2Int8 convert string Slice 2 int8
func StringSlice2Int8(bs []string) byte {
	return String2Int8(strings.Join(bs, ""), 2)
}

// CheckLrc check lrc data
func CheckLrc(d []byte) bool {
	rowdata := d[:len(d)-1]
	lrcdata := d[len(d)-1]

	c := CountLrc(&rowdata)
	if c == lrcdata {
		return true
	}
	return false
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

// IPUint2String change ip int64 data to string format
func IPUint2String(ipnr uint) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ipnr>>24)&0xFF, (ipnr>>16)&0xFF, (ipnr>>8)&0xFF, ipnr&0xFF)
}

// IPInt642String change ip int64 data to string format
func IPInt642String(ipnr int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ipnr)&0xFF, (ipnr>>8)&0xFF, (ipnr>>16)&0xFF, (ipnr>>24)&0xFF)
}

// IPInt642Bytes change ip int64 data to string format
func IPInt642Bytes(ipnr int64) []byte {
	return []byte{byte((ipnr) & 0xFF), byte((ipnr >> 8) & 0xFF), byte((ipnr >> 16) & 0xFF), byte((ipnr >> 24) & 0xFF)}
}

// IPUint2Bytes change ip int64 data to string format
func IPUint2Bytes(ipnr int64) []byte {
	return []byte{byte((ipnr >> 24) & 0xFF), byte((ipnr >> 16) & 0xFF), byte((ipnr >> 8) & 0xFF), byte((ipnr) & 0xFF)}
}

// IP2Uint change ip string data to int64 format
func IP2Uint(ipnr string) uint {
	// ex := errors.New("wrong ip address format")
	bits := strings.Split(ipnr, ".")
	if len(bits) != 4 {
		return 0
	}
	var intip uint
	for k, v := range bits {
		i, ex := strconv.Atoi(v)
		if ex != nil || i > 255 || i < 0 {
			return 0
		}
		intip += uint(i) << uint(8*(3-k))
	}
	return intip
}

// IP2Int64 change ip string data to int64 format
func IP2Int64(ipnr string) int64 {
	// ex := errors.New("wrong ip address format"
	bits := strings.Split(ipnr, ".")
	if len(bits) != 4 {
		return 0
	}
	var intip uint
	for k, v := range bits {
		i, ex := strconv.Atoi(v)
		if ex != nil || i > 255 || i < 0 {
			return 0
		}
		intip += uint(i) << uint(8*(k))
	}
	return int64(intip)
}

// IsExist file is exist or not
func IsExist(p string) bool {
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}

// GetExecDir get current file path
func GetExecDir() string {
	a, _ := os.Executable()
	execdir := filepath.Dir(a)
	if strings.Contains(execdir, "go-build") {
		execdir, _ = filepath.Abs(".")
	}
	return execdir
}

//SplitDateTime SplitDateTime
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
		String2Int8(dd[1], 10),
		String2Int8(dd[2], 10),
		String2Int8(tt[0], 10),
		String2Int8(tt[1], 10),
		String2Int8(tt[2], 10),
		byte(tm.Weekday())
}

// Stamp2Time convert stamp to datetime string
func Stamp2Time(t int64, fmt ...string) string {
	var f string
	if len(fmt) > 0 {
		f = fmt[0]
	} else {
		f = "2006-01-02 15:04:05"
	}
	tm := time.Unix(t, 0)
	return tm.Format(f)
}

// Time2Stampf 可根据制定的时间格式和时区转换为当前时区的Unix时间戳
//	s：时间字符串
//  fmt：时间格式
//  year：2006，month：01，day：02
//  hour：15，minute：04，second：05
//  tz：0～12,超范围时使用本地时区
func Time2Stampf(s, fmt string, tz float32) int64 {
	if fmt == "" {
		fmt = "2006-01-02 15:04:05"
	}
	if tz > 12 || tz < 0 {
		_, t := time.Now().Zone()
		tz = float32(t / 3600)
	}
	var loc *time.Location
	loc = time.FixedZone("", int((time.Duration(tz) * time.Hour).Seconds()))
	tm, ex := time.ParseInLocation(fmt, s, loc)
	if ex != nil {
		return 0
	}
	return tm.Unix()
}

// Time2Stamp convert datetime string to stamp
func Time2Stamp(s string) int64 {
	return Time2Stampf(s, "", 8)
}

// Time2StampNB 电信NB平台数据时间戳转为本地unix时间戳
func Time2StampNB(s string) int64 {
	return Time2Stampf(s, "20060102T150405Z", 0)
}

// SwitchStamp switch stamp format between unix and c#
func SwitchStamp(t int64) int64 {
	y := int64(621356256000000000)
	z := int64(10000000)
	if t > y {
		return (t - y) / z
	}
	return t*z + y
}

// Byte2Bytes int8 to bytes
func Byte2Bytes(v byte, reverse bool) []byte {
	s := fmt.Sprintf("%08b", v)
	if reverse {
		s = ReverseString(s)
	}
	b := make([]byte, 0)
	for _, v := range s {
		if v == 48 {
			b = append(b, 0)
		} else {
			b = append(b, 1)
		}
	}
	return b
}

// Byte2Int32s int8 to int32 list
func Byte2Int32s(v byte, reverse bool) []int32 {
	s := fmt.Sprintf("%08b", v)
	if reverse {
		s = ReverseString(s)
	}
	b := make([]int32, 0)
	for _, v := range s {
		if v == 48 {
			b = append(b, 0)
		} else {
			b = append(b, 1)
		}
	}
	return b
}

// Bcd2Int8 bcd to int
func Bcd2Int8(v byte) byte {
	return ((v&0xf0)>>4)*10 + (v & 0x0f)
}

// Int82Bcd int to bcd
func Int82Bcd(v byte) byte {
	return ((v / 10) << 4) | (v % 10)
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
	for _, v := range []byte(s) {
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
		x := String2Int8(string(y[0])+string(y[1]), 0)
		z := y[2:]
		for i := len(z) - 1; i >= 0; i-- {
			if z[i] >= x {
				ns.WriteByte(z[i] - x)
			} else {
				ns.WriteByte(byte(int(z[i]) + 256 - int(x)))
			}
		}
		return ReverseString(string(DoZlibUnCompress(ns.Bytes())))
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
			ns.WriteString(string(int(v) + 32))
		} else if v >= 97 && v <= 122 {
			ns.WriteString(string(int(v) - 32))
		} else {
			ns.WriteString(string(v))
		}
	}
	return ns.String()
}

// VersionInfo show something
//  args:
// 	p: program name
// 	v: program version
// 	gv: golang version
// 	bd: build datetime
// 	pl: platform info
// 	a: auth name
func VersionInfo(p, v, gv, bd, pl, a string) string {
	return fmt.Sprintf("\n%s\r\nVersion:\t%s\r\nGo version:\t%s\r\nBuild date:\t%s\r\nBuild OS:\t%s\r\nCode by:\t%s\r\nStart with:\t%s", p, v, gv, pl, bd, a, strings.Join(os.Args[1:], " "))
}

// WriteVersionInfo write version info to .ver file
//  args:
// 	p: program name
// 	v: program version
// 	gv: golang version
// 	bd: build datetime
// 	pl: platform info
// 	a: auth name
func WriteVersionInfo(p, v, gv, bd, pl, a string) {
	fn, _ := os.Executable()
	f, _ := os.OpenFile(fmt.Sprintf("%s.ver", fn), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0444)
	defer f.Close()
	f.WriteString(fmt.Sprintf("\n%s\r\nVersion:\t%s\r\nGo version:\t%s\r\nBuild date:\t%s\r\nBuild OS:\t%s\r\nCode by:\t%s\r\nStart with:\t%s", p, v, gv, pl, bd, a, strings.Join(os.Args[1:], " ")))
}

// CalculateSecurityCode calculate security code
//  args:
//	t: calculate type "h"-按小时计算，当分钟数在偏移值范围内时，同时计算前后一小时的值，"m"-按分钟计算，同时计算前后偏移量范围内的值
//	salt: 拼接用字符串
//	offset: 偏移值，范围0～59
//  return:
//	32位小写md5码切片
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
		rs.WriteByte(byte(rand.Int31n(256) + 1))
	}
	return rs.Bytes()
}

// GetRandomString 生成随机字符串
func GetRandomString(l int64) string {
	str := "!#%&()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}"
	bb := []byte(str)
	var rs bytes.Buffer
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := int64(0); i < l; i++ {
		rs.WriteByte(bb[rand.Intn(len(bb))])
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
	jsonStr, err := json.MarshalToString(pb)
	if err != nil {
		return ""
	}
	return jsonStr
}

// JSON2PB json字符串转pb2格式
func JSON2PB(js string, pb interface{}) error {
	err := json.Unmarshal([]byte(js), &pb)
	return err
}

// Uint642Bytes 长整形转换字节数组（8位），bigOrder==true，高位在前
func Uint642Bytes(i uint64, bigOrder bool) []byte {
	var buf = make([]byte, 8)
	if bigOrder {
		binary.BigEndian.PutUint64(buf, i)
	} else {
		binary.LittleEndian.PutUint64(buf, i)
	}
	return buf
}

// Int642Bytes 无符号长整形转换字节数组（8位），bigOrder==true，高位在前
func Int642Bytes(i int64, bigOrder bool) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	if bigOrder {
		binary.Write(bytesBuffer, binary.BigEndian, &i)
	} else {
		binary.Write(bytesBuffer, binary.LittleEndian, &i)
	}
	return bytesBuffer.Bytes()
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

// Bytes2Float64 字节数组转双精度浮点，bigOrder==true,高位在前
func Bytes2Float64(b []byte, bigOrder bool) float64 {
	return math.Float64frombits(Bytes2Uint64(b, bigOrder))
}

// Bytes2Float32 字节数组转单精度浮点，bigOrder==true,高位在前
func Bytes2Float32(b []byte, bigOrder bool) float32 {
	return math.Float32frombits(uint32(Bytes2Uint64(b, bigOrder)))
}

// Imgfile2Base64 图片转base64
func Imgfile2Base64(s string) (string, error) {
	f, err := ioutil.ReadFile(s)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(f), nil
}

// Base64Imgfile base64码保存为图片
func Base64Imgfile(b, f string) error {
	a, err := base64.StdEncoding.DecodeString(b)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(f, a, 0666)
}

// SplitStringWithLen 按制定长度分割字符串
// 	s-原始字符串
//	l-切割长度
func SplitStringWithLen(s string, l int) []string {
	rs := []rune(s)
	var ss = make([]string, 0)
	xs := ""
	for k, v := range rs {
		xs = xs + string(v)
		if (k+1)%l == 0 {
			ss = append(ss, xs)
			xs = ""
		}
	}
	if len(xs) > 0 {
		ss = append(ss, xs)
	}
	return ss
}

// HexString2Bytes 转换hexstring为字节数组
// 	s-hexstring（11aabb）
//	bigorder-是否高位在前
//	false低位在前
func HexString2Bytes(s string, bigorder bool) []byte {
	if len(s)%2 == 1 {
		s = "0" + s
	}
	ss := SplitStringWithLen(s, 2)
	var b = make([]byte, len(ss))
	if bigorder {
		for k, v := range ss {
			b[k] = String2Int8(v, 16)
		}
	} else {
		c := 0
		for i := len(ss) - 1; i >= 0; i-- {
			b[c] = String2Int8(ss[i], 16)
			c++
		}
	}
	return b
}

// Bytes2Uint64 字节数组转换为uint64
// 	b-字节数组
//	bigorder-是否高位在前
//	false低位在前
func Bytes2Uint64(b []byte, bigorder bool) uint64 {
	s := ""
	for _, v := range b {
		if bigorder {
			s = s + fmt.Sprintf("%02x", v)
		} else {
			s = fmt.Sprintf("%02x", v) + s
		}
	}
	u, _ := strconv.ParseUint(s, 16, 64)
	return u
}

// Bytes2Int64 字节数组转换为uint64
// 	b-字节数组
// 	bigorder-是否高位在前
//	false低位在前
func Bytes2Int64(b []byte, bigorder bool) int64 {
	s := ""
	for _, v := range b {
		if bigorder {
			s = s + fmt.Sprintf("%02x", v)
		} else {
			s = fmt.Sprintf("%02x", v) + s
		}
	}
	u, _ := strconv.ParseInt(s, 16, 64)
	return u
}

// GetServerTLSConfig 获取https配置
//	certfile: 服务端证书
// 	keyfile: 服务端key
// 	clientca: 双向验证时客户端根证书
func GetServerTLSConfig(certfile, keyfile, clientca string) (*tls.Config, error) {
	tc := &tls.Config{}
	cliCrt, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return nil, err
	}
	tc.Certificates = []tls.Certificate{cliCrt}
	caCrt, err := ioutil.ReadFile(clientca)
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
// 	certfile: 双向验证时客户端证书
// 	keyfile: 双向验证时客户端key
// 	rootca: 服务端根证书
func GetClientTLSConfig(certfile, keyfile, rootca string) (*tls.Config, error) {
	tc := &tls.Config{}
	var err error
	caCrt, err := ioutil.ReadFile(rootca)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if pool.AppendCertsFromPEM(caCrt) {
		tc.RootCAs = pool
		tc.InsecureSkipVerify = false
	} else {
		tc.InsecureSkipVerify = true
	}

	cliCrt, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return tc, nil
	}
	tc.Certificates = []tls.Certificate{cliCrt}
	return tc, nil
}

// EncodeUTF16BE 将字符串编码成utf16be的格式，用于cdma短信发送
func EncodeUTF16BE(s string) []byte {
	a := utf16.Encode([]rune(s))
	var b bytes.Buffer
	for _, v := range a {
		b.Write([]byte{byte(v >> 8), byte(v)})
	}
	return b.Bytes()
}

// TrimString 去除字符串末尾的空格，\r\n
func TrimString(s string) string {
	r := strings.NewReplacer("\r", "", "\n", "", "\000", "")
	return r.Replace(strings.TrimSpace(s))
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
func ZIPFile(d, s string, delold bool) {
	// defer func() {
	// 	if delold {
	// 		os.Remove(filepath.Join(d, s))
	// 	}
	// }()
	zfile := filepath.Join(d, s+".zip")
	ofile := filepath.Join(d, s)

	newZipFile, err := os.Create(zfile)
	if err != nil {
		return
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	zipfile, err := os.Open(ofile)
	if err != nil {
		return
	}
	defer zipfile.Close()
	info, err := zipfile.Stat()
	if err != nil {
		return
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return
	}
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return
	}
	if _, err = io.Copy(writer, zipfile); err != nil {
		return
	}
	if delold {
		os.Remove(filepath.Join(d, s))
	}
}

// GPS2DFM 经纬度转度分秒
func GPS2DFM(l float64) (int, int, float64) {
	du, x := math.Modf(l)
	fen, y := math.Modf(x * 60)
	return int(du), int(fen), y * 60
}

// DFM2GPS 度分秒转经纬度
func DFM2GPS(du, fen int, miao float64) float64 {
	return float64(du) + float64(fen)/60 + miao/360000
}

// Float642BcdBytes 浮点转bcd字节数组（小端序）
// 	v：十进制浮点数值
// 	f：格式化的字符串，如"%07.03f","%03.0f"
func Float642BcdBytes(v float64, f string) []byte {
	s := strings.ReplaceAll(fmt.Sprintf(f, math.Abs(v)), ".", "")
	var b bytes.Buffer
	if len(s)%2 != 0 {
		s = "0" + s
	}
	for i := len(s); i > 1; i -= 2 {
		if i == 2 {
			if v > 0 {
				b.WriteByte(String2Int8(s[i-2:i], 16))
			} else {
				b.WriteByte(String2Int8(s[i-2:i], 16) + 0x80)
			}
		} else {
			b.WriteByte(String2Int8(s[i-2:i], 16))
		}
	}
	return b.Bytes()
}

// BcdBytes2Float64 bcd数组转浮点(小端序)
// 	b:bcd数组
// 	d：小数位数
// 	Unsigned：无符号的
func BcdBytes2Float64(b []byte, decimal int, unsigned bool) float64 {
	var negative = false
	var s string
	for k, v := range b {
		if k == len(b)-1 { // 最后一位，判正负
			if !unsigned {
				if v >= 128 {
					v = v - 0x80
					negative = true
				}
			}
		}
		s = fmt.Sprintf("%02x", v) + s
	}
	s = s[:len(s)-decimal] + "." + s[len(s)-decimal:]
	f, _ := strconv.ParseFloat(s, 16)
	if negative {
		f = f * -1
	}
	return f
}

// Bcd2STime bcd转hh*60+mm
func Bcd2STime(b []byte) int32 {
	return String2Int32(fmt.Sprintf("%02x", b[0]), 10)*60 + String2Int32(fmt.Sprintf("%02x", b[1]), 10)
}

// STime2Bcd hh*60+mm转BCD
func STime2Bcd(t int32) []byte {
	return []byte{Bcd2Int8(byte(t / 60)), Bcd2Int8(byte(t % 60))}
}

// Byte2SignedInt32 转有符号整型
func Byte2SignedInt32(b byte) int32 {
	if b > 127 {
		return (int32(b) - 128) * -1
	}
	return int32(b)
}

// SignedInt322Byte 有符号整型转byte
func SignedInt322Byte(i int32) byte {
	if i > 127 || i < -127 {
		return byte(0)
	}
	if i < 0 {
		return byte(i*-1 + 128)
	}
	return byte(i)
}

// BcdDT2Stamp bcd时间戳转unix
func BcdDT2Stamp(d []byte) int64 {
	var f = "0601021504"
	if len(d) == 6 {
		f = "060102150405"
	}
	return Time2Stampf(fmt.Sprintf("%d", int(BcdBytes2Float64(d, 0, true))), f, 8)
}

// Stamp2BcdDT unix时间戳转bcd,6字节，第一字节为秒
func Stamp2BcdDT(dt int64) []byte {
	return Float642BcdBytes(String2Float64(Stamp2Time(dt, "060102150405")), "%12.0f")
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
