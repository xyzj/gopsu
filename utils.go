package gopsu

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"container/list"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
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

	"github.com/google/uuid"

	"github.com/golang/snappy"
	"github.com/pierrec/lz4"
	// _ "github.com/go-sql-driver/mysql"
)

const (
	// OSNAME from runtime
	OSNAME = runtime.GOOS
	// OSARCH from runtime
	OSARCH = runtime.GOARCH
)
const (
	CryptoMD5 = iota
	CryptoSHA256
	CryptoSHA512
	CryptoAES128CBC
	CryptoAES128CFB
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

// GetNewCryptoWorker 获取新的序列化或加密管理器
func GetNewCryptoWorker(cryptoType byte) *CryptoWorker {
	h := &CryptoWorker{
		cryptoType: cryptoType,
		cryptoKey:  []byte("d64c7be2-3644-11e9-a13f-aaaa000ef3c9"),
	}
	ctx := md5.New()
	ctx.Write(h.cryptoKey)
	switch cryptoType {
	case CryptoMD5:
		h.cryptoHash = md5.New()
	case CryptoSHA256:
		h.cryptoHash = sha256.New()
	case CryptoSHA512:
		h.cryptoHash = sha512.New()
	case CryptoAES128CBC:
		h.cryptoBlock, _ = aes.NewCipher([]byte(hex.EncodeToString(ctx.Sum(nil)))[:16])
		h.cryptoIV = make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(crand.Reader, h.cryptoIV); err != nil {
			h.cryptoIV = []byte(GetRandomString(16))
		}
		h.cryptoCBCEncrypter = cipher.NewCBCEncrypter(h.cryptoBlock, h.cryptoIV)
		h.cryptoCBCDecrypter = cipher.NewCBCDecrypter(h.cryptoBlock, h.cryptoIV)
	case CryptoAES128CFB:
		h.cryptoBlock, _ = aes.NewCipher([]byte(hex.EncodeToString(ctx.Sum(nil)))[:16])
		h.cryptoIV = make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(crand.Reader, h.cryptoIV); err != nil {
			h.cryptoIV = []byte(GetRandomString(16))
		}
		h.cryptoCFBEncrypter = cipher.NewCFBEncrypter(h.cryptoBlock, h.cryptoIV)
		h.cryptoCFBDecrypter = cipher.NewCFBDecrypter(h.cryptoBlock, h.cryptoIV)
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
func (h *CryptoWorker) SetKey(k, iv string) {
	ctx := md5.New()
	ctx.Reset()
	ctx.Write([]byte(iv))
	switch h.cryptoType {
	case CryptoAES128CBC:
		ctx.Write([]byte(k))
		h.cryptoBlock, _ = aes.NewCipher([]byte(hex.EncodeToString(ctx.Sum(nil)))[:16])
		ctx.Reset()
		ctx.Write([]byte(iv))
		h.cryptoIV = []byte(hex.EncodeToString(ctx.Sum(nil)))[:16]
		h.cryptoCBCEncrypter = cipher.NewCBCEncrypter(h.cryptoBlock, h.cryptoIV)
		h.cryptoCBCDecrypter = cipher.NewCBCDecrypter(h.cryptoBlock, h.cryptoIV)
	case CryptoAES128CFB:
		ctx.Write([]byte(k))
		h.cryptoBlock, _ = aes.NewCipher([]byte(hex.EncodeToString(ctx.Sum(nil)))[:16])
		ctx.Reset()
		ctx.Write([]byte(iv))
		h.cryptoIV = []byte(hex.EncodeToString(ctx.Sum(nil)))[:16]
		h.cryptoCBCEncrypter = cipher.NewCBCEncrypter(h.cryptoBlock, h.cryptoIV)
		h.cryptoCBCDecrypter = cipher.NewCBCDecrypter(h.cryptoBlock, h.cryptoIV)
	default:
	}
}

// Encrypt 加密
func (h *CryptoWorker) Encrypt(s string) string {
	switch h.cryptoType {
	case CryptoAES128CBC:
		content := pkcs5Padding([]byte(s), h.cryptoBlock.BlockSize())
		crypted := make([]byte, len(content))
		h.cryptoCBCEncrypter.CryptBlocks(crypted, content)
		return base64.StdEncoding.EncodeToString(crypted)
	case CryptoAES128CFB:
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
	if x := 4 - len(s)%4; x != 4 {
		for i := 0; i < x; i++ {
			s += "="
		}
	}

	msg, _ := base64.StdEncoding.DecodeString(s)
	switch h.cryptoType {
	case CryptoAES128CBC:
		decrypted := make([]byte, len(msg))
		h.cryptoCBCDecrypter.CryptBlocks(decrypted, msg)
		return string(pkcs5Unpadding(decrypted))
	case CryptoAES128CFB:
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
	// ArchiveLZ4 lz4压缩/解压缩
	ArchiveLZ4
	// ArchiveLZ4HC lz4hc压缩/解压缩
	ArchiveLZ4HC
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
	lz4Reader        *lz4.Reader
	lz4Writer        *lz4.Writer
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
	case ArchiveLZ4:
		a.lz4Reader = lz4.NewReader(a.in)
		a.lz4Writer = lz4.NewWriter(&a.code)
		a.lz4Reader.Header = lz4.Header{
			CompressionLevel: 6,
		}
		a.lz4Writer.Header = lz4.Header{
			CompressionLevel: 6,
		}
	case ArchiveLZ4HC:
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
	case ArchiveLZ4HC:
		var b = make([]byte, len(src))
		di, err := lz4.CompressBlockHC(src, b, 0)
		if err == nil {
			a.code.Write(b[:di])
		}
	case ArchiveLZ4:
		a.lz4Writer.Reset(&a.code)
		a.lz4Writer.Write(src)
		a.lz4Writer.Close()
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
	case ArchiveLZ4HC:
		var dst []byte
		var count = 300
		for {
			dst = make([]byte, len(src)*count)
			di, err := lz4.UncompressBlock(src, dst)
			if err == nil {
				a.decode.Write(dst[:di])
				break
			} else {
				if err == lz4.ErrInvalidSourceShortBuffer {
					count += 10
				} else {
					break
				}
			}
		}
	case ArchiveLZ4:
		a.in.Reset(src)
		a.lz4Reader = lz4.NewReader(a.in)
		a.lz4Reader.Header = lz4.Header{
			CompressionLevel: 6,
		}
		buf := make([]byte, len(src)*300)
		for {
			n, err := a.lz4Reader.Read(buf)
			if err != nil || err == io.EOF || n == 0 {
				break
			}
			if n > 0 {
				a.decode.Write(buf[:n])
			}
		}
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

// CompressData 使用gzip，zlib，lz4压缩数据
// lz4hc 目前无法跨语言使用
func CompressData(src []byte, t byte) []byte {
	var in bytes.Buffer
	switch t {
	case ArchiveGZip:
		w := gzip.NewWriter(&in)
		w.Write(src)
		w.Close()
	case ArchiveLZ4HC:
		var b = make([]byte, len(src))
		di, err := lz4.CompressBlockHC(src, b, 0)
		if err == nil {
			in.Write(b[:di])
		}
	case ArchiveLZ4:
		w := lz4.NewWriter(&in)
		w.Header = lz4.Header{
			CompressionLevel: 6,
		}
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

// UncompressData 使用gzip，zlib，lz4解压缩数据
func UncompressData(src []byte, t byte, dstlen ...interface{}) []byte {
	var out bytes.Buffer
	switch t {
	case ArchiveGZip:
		b := bytes.NewReader(src)
		r, _ := gzip.NewReader(b)
		io.Copy(&out, r)
	case ArchiveLZ4:
		b := bytes.NewReader(src)
		r := lz4.NewReader(b)
		r.Header = lz4.Header{
			CompressionLevel: 6,
		}
		buf := make([]byte, len(src)*300)
		for {
			n, err := r.Read(buf)
			if err != nil || err == io.EOF || n == 0 {
				break
			}
			if n > 0 {
				out.Write(buf[:n])
			}
		}
	case ArchiveLZ4HC:
		var dst []byte
		var count = 500
		if len(dstlen) > 0 {
			if value, ok := dstlen[0].(int); ok == true {
				count = value/len(src) + 1
			}
		}
		// RETRY:
		for {
			dst = make([]byte, len(src)*count)
			di, err := lz4.UncompressBlock(src, dst)
			if err == nil {
				out.Write(dst[:di])
				break
			} else {
				if err == lz4.ErrInvalidSourceShortBuffer {
					count += 10
				} else {
					break
				}
			}
		}
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
	return uuid.New().String()
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
	} else {
		return len(arr.TwoDimensional)
	}
}

func (arr *StringSliceSort) Swap(i, j int) {
	if len(arr.OneDimensional) > 0 {
		arr.OneDimensional[i], arr.OneDimensional[j] = arr.OneDimensional[j], arr.OneDimensional[i]
	} else {
		arr.TwoDimensional[i], arr.TwoDimensional[j] = arr.TwoDimensional[j], arr.TwoDimensional[i]
	}
}

func (arr *StringSliceSort) Less(i, j int) bool {
	if arr.Order == "desc" {
		if len(arr.OneDimensional) > 0 {
			return arr.OneDimensional[i] > arr.OneDimensional[j]
		} else {
			arr1 := arr.TwoDimensional[i]
			arr2 := arr.TwoDimensional[j]
			if arr.Idx > len(arr.TwoDimensional[0]) {
				arr.Idx = 0
			}
			return arr1[arr.Idx] > arr2[arr.Idx]
		}
	} else {
		if len(arr.OneDimensional) > 0 {
			return arr.OneDimensional[i] < arr.OneDimensional[j]
		} else {
			arr1 := arr.TwoDimensional[i]
			arr2 := arr.TwoDimensional[j]
			if arr.Idx > len(arr.TwoDimensional[0]) {
				arr.Idx = 0
			}
			return arr1[arr.Idx] < arr2[arr.Idx]
		}
	}
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
			if y, ex := strconv.ParseInt(strings.TrimSpace(v), 10, 0); ex != nil {
				return nil, ex
			} else {
				lstAddr = append(lstAddr, y)
			}
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
		if a, ex := regexp.Match(regipwithport, []byte(ip)); ex != nil {
			return false
		} else {
			return a
		}
	} else {
		if a, ex := regexp.Match(regip, []byte(ip)); ex != nil {
			return false
		} else {
			return a
		}
	}
}

// MakeRuntimeDirs make conf,log,cache dirs
//  Args：
//	rootpath： 输入路径
//  return：
// 	conf，log，cache三个文件夹的完整路径
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
	var z = make([]byte, 0)
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
	z = append(z, byte(crc16lo), byte(crc16hi))
	return z
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
//  fmt：
//  year：2006
//  month：01
//  day：02
//  hour：15
//  minute：04
//  second：05
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
	} else {
		return ""
	}
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
	} else {
		return ""
	}
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
		// println(v, string(v))
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
	return fmt.Sprintf("\n%s\r\nVersion:\t%s\r\nGo version:\t%s\r\nBuild date:\t%s\r\nBuild OS:\t%s\r\nCode by:\t%s", p, v, gv, pl, bd, a)
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
	f.WriteString(fmt.Sprintf("\n%s\r\nVersion:\t%s\r\nGo version:\t%s\r\nBuild date:\t%s\r\nBuild OS:\t%s\r\nCode by:\t%s\r\n", p, v, gv, pl, bd, a))
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

// GetRandomString 生成随机字符串
func GetRandomString(l int64) string {
	str := "!#$%&()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}"
	bb := []byte(str)
	var rs bytes.Buffer
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := int64(0); i < l; i++ {
		rs.WriteByte(bb[r.Intn(len(bb))])
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

// PB2Json pb2格式转换为json字符串
func PB2Json(pb interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(pb)
	return jsonBytes, err
}

// Json2PB json字符串转pb2格式
func Json2PB(js string, pb interface{}) error {
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

// UInt642Bytes 无符号长整形转换字节数组（8位），bigOrder==true，高位在前
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
func Bytes2Int64(b []byte, bigOrder bool) int64 {
	var l = len(b)
	switch l {
	case 1:
		var tmp int8
		bytesBuffer := bytes.NewBuffer(b)
		if bigOrder {
			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
		} else {
			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
		}
		return int64(tmp)
	case 2:
		var tmp int16
		bytesBuffer := bytes.NewBuffer(b)
		if bigOrder {
			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
		} else {
			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
		}
		return int64(tmp)
	case 3, 4:
		var tmp int32
		bytesBuffer := bytes.NewBuffer(b)
		if bigOrder {
			if l == 3 {
				b = append([]byte{0}, b...)
			}
			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
		} else {
			if l == 3 {
				b = append(b, 0)
			}
			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
		}
		return int64(tmp)
	case 5, 6, 7, 8:
		var tmp int64
		bytesBuffer := bytes.NewBuffer(b)
		if bigOrder {
			if l < 8 {
				bb := make([]byte, 8-l)
				b = append(bb, b...)
			}
			binary.Read(bytesBuffer, binary.BigEndian, &tmp)
		} else {
			if l < 8 {
				bb := make([]byte, 8-l)
				b = append(b, bb...)
			}
			binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
		}
		return int64(tmp)
	}
	return 0
}

// Bytes2Uint64 字节数组转换为uint64，bigOrder==true,高位在前
func Bytes2Uint64(b []byte, bigOrder bool) uint64 {
	var l int
	if len(b) > 8 {
		l = 0
	} else {
		l = 8 - len(b)
	}
	var bb = make([]byte, l)
	if bigOrder {
		bb = append(bb, b...)
		b = bb
	} else {
		b = append(b, bb...)
	}
	if bigOrder {
		return binary.BigEndian.Uint64(b)
	} else {
		return binary.LittleEndian.Uint64(b)
	}
}

// Bytes2Float64 字节数组转双精度浮点，bigOrder==true,高位在前
func Bytes2Float64(b []byte, bigOrder bool) float64 {
	return math.Float64frombits(Bytes2Uint64(b, bigOrder))
}

// Bytes2Float32 字节数组转单精度浮点，bigOrder==true,高位在前
func Bytes2Float32(b []byte, bigOrder bool) float32 {
	return math.Float32frombits(uint32(Bytes2Uint64(b, bigOrder)))
}
