package ginmiddleware

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/xyzj/gopsu"
)

var json = jsoniter.Config{}.Froze()
var md5worker = gopsu.GetNewCryptoWorker(gopsu.CryptoMD5)

type ginLogger struct {
	fno       *os.File     // 文件日志
	fname     string       // 日志文件名
	fileIndex byte         // 文件索引号
	expired   int64        // 日志文件过期时长
	flock     sync.RWMutex // 同步锁
	nameOld   string       // 旧日志文件名
	nameNow   string       // 当前日志文件名
	fileHour  int          // 旧时间戳
	fileDay   int          // 日期戳
	pathOld   string       // 写入用日志路径
	pathNow   string       // 当前日志路径
	logDir    string       // 日志文件夹
	maxDays   int          // 文件有效时间
	out       io.Writer    // io写入
	err       error        // 错误信息
	enablegz  bool         // 是否允许gzip压缩旧日志文件
}

// LoggerWithRolling 滚动日志
// logdir: 日志存放目录。
// filename：日志文件名。
// maxdays：日志文件最大保存天数。
func LoggerWithRolling(logdir, filename string, maxdays int) gin.HandlerFunc {
	t := time.Now()
	// 初始化
	f := &ginLogger{
		logDir:   logdir,
		fname:    filename,
		expired:  int64(maxdays)*24*60*60 - 10,
		maxDays:  maxdays,
		fileHour: t.Hour(),
		fileDay:  t.Day(),
		pathOld:  filepath.Join(logdir, fmt.Sprintf("%s.current.log", filename)),
		enablegz: true,
	}
	// 搜索最后一个文件名
	for i := byte(255); i > 0; i-- {
		if gopsu.IsExist(filepath.Join(f.logDir, fmt.Sprintf("%s.%v.%d.log", filename, t.Format(gopsu.FileTimeFormat), i))) {
			f.fileIndex = i
		}
	}
	f.pathNow = filepath.Join(logdir, fmt.Sprintf("%s.%v.%d.log", filename, t.Format(gopsu.FileTimeFormat), f.fileIndex))
	// 创建新日志
	f.newFile()
	// 设置io
	gin.DefaultWriter = f.out
	gin.DefaultErrorWriter = f.out

	return func(c *gin.Context) {
		// 检查是否需要切分文件
		if f.rollingFile() {
			gin.DefaultWriter = f.out
			gin.DefaultErrorWriter = f.out
		}
		start := time.Now()

		token := c.GetHeader("User-Token")
		if len(token) == 36 {
			token = md5worker.Hash([]byte(token))
		}

		c.Next()
		path := c.Request.RequestURI
		param := &gin.LogFormatterParams{
			Request: c.Request,
			Keys:    c.Keys,
		}
		// Stop timer
		param.TimeStamp = time.Now()
		param.Latency = param.TimeStamp.Sub(start)
		param.ClientIP = c.ClientIP()
		param.Method = c.Request.Method
		param.StatusCode = c.Writer.Status()
		param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()
		param.BodySize = c.Writer.Size()
		if body, ok := c.Params.Get("_body"); ok {
			path += "|" + body
		}
		if len(token) == 32 {
			path = "(" + token + ")" + path
		}
		param.Path = path

		var s string
		if len(param.Keys) == 0 {
			s = fmt.Sprintf("%v |%3d| %-10s | %-15s|%-4s %s",
				param.TimeStamp.Format(gopsu.ShortTimeFormat),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
			)
		} else {
			jsn, _ := json.Marshal(param.Keys)
			s = fmt.Sprintf("%v |%3d| %-10s | %-15s|%-4s %s ▸%s",
				param.TimeStamp.Format(gopsu.ShortTimeFormat),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
				jsn,
			)
		}
		if param.ErrorMessage != "" {
			s += " #" + param.ErrorMessage
		}
		go func() {
			defer func() { recover() }()
			fmt.Fprintln(f.out, s)
		}()
	}
}

// 检查文件大小,返回是否需要切分文件
func (f *ginLogger) rolledWithFileSize() bool {
	if f.fileHour == time.Now().Hour() {
		return false
	}
	f.fileHour = time.Now().Hour()
	fs, ex := os.Stat(f.pathNow)
	if ex == nil {
		if fs.Size() > 1048576000 {
			if f.fileIndex >= 255 {
				f.fileIndex = 0
			} else {
				f.fileIndex++
			}
			return true
		}
	}
	return false
}

// 按日期切分文件
func (f *ginLogger) rollingFile() bool {
	f.flock.Lock()
	defer f.flock.Unlock()

	t := time.Now()
	f.rolledWithFileSize()
	f.nameNow = fmt.Sprintf("%s.%v.%d.log", f.fname, t.Format(gopsu.FileTimeFormat), f.fileIndex)
	// 比对文件名，若不同则重新设置io
	if f.nameNow == f.nameOld {
		return false
	}
	// 关闭旧fno
	f.fno.Close()
	// 创建新日志
	f.newFile()
	// 清理旧日志
	f.cleanFile()

	return true
}

// 压缩旧日志
func (f *ginLogger) zipFile(s string) {
	if !f.enablegz || len(s) == 0 || !gopsu.IsExist(filepath.Join(f.logDir, s)) {
		return
	}
	go func(s string) {
		err := gopsu.ZIPFile(f.logDir, s, true)
		if err != nil {
			fmt.Fprintln(f.out, "zip log file error: "+s+" "+err.Error())
			return
		}
		// 删除已压缩的旧日志
		err = os.Remove(filepath.Join(f.logDir, s))
		if err != nil {
			fmt.Fprintln(f.out, "gin del old file error: "+s+" "+err.Error())
			// ioutil.WriteFile(fmt.Sprintf("logcrash.%d.log", time.Now().Unix()), []byte("del old file:"+s+" "+err.Error()), 0664)
		}
	}(s)
}

// 清理旧日志
func (f *ginLogger) cleanFile() {
	// 若未设置超时，则不清理
	if f.expired == 0 {
		return
	}
	go func() {
		defer func() { recover() }()

		// 遍历文件夹
		lstfno, ex := ioutil.ReadDir(f.logDir)
		if ex != nil {
			ioutil.WriteFile("ginlogerr.log", []byte(fmt.Sprintf("clear log files error: %s", ex.Error())), 0644)
			return
		}
		t := time.Now()
		for _, fno := range lstfno {
			if fno.IsDir() || !strings.Contains(fno.Name(), f.fname) { // 忽略目录，不含日志名的文件，以及当前文件
				continue
			}
			// 比对文件生存期
			if t.Unix()-fno.ModTime().Unix() >= f.expired {
				os.Remove(filepath.Join(f.logDir, fno.Name()))
			}
		}
	}()
}

// 创建新日志文件
func (f *ginLogger) newFile() {
	t := time.Now()
	if f.fileDay != t.Day() {
		f.fileDay = t.Day()
		f.fileIndex = 0
	}
	// 直接写入当日日志
	f.nameNow = fmt.Sprintf("%s.%v.%d.log", f.fname, t.Format(gopsu.FileTimeFormat), f.fileIndex)
	f.pathNow = filepath.Join(f.logDir, f.nameNow)
	f.pathOld = f.pathNow
	if f.fname == "" {
		f.out = io.MultiWriter(os.Stdout)
	} else {
		// 打开文件
		f.fno, f.err = os.OpenFile(f.pathOld, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
		if f.err != nil {
			ioutil.WriteFile("ginlogerr.log", []byte("Log file open error: "+f.err.Error()), 0644)
			f.out = io.MultiWriter(os.Stdout)
		} else {
			if gin.Mode() == "debug" {
				f.out = io.MultiWriter(f.fno, os.Stdout)
			} else {
				f.out = io.MultiWriter(f.fno)
			}
		}
		// 判断是否压缩旧日志
		if f.enablegz {
			f.zipFile(f.nameOld)
		}
	}
	f.nameOld = f.nameNow
}
