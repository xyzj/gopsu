package ginmiddleware

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/xyzj/gopsu"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type ginLogger struct {
	fno      *os.File     // 文件日志
	fname    string       // 日志文件名
	fexpired int64        // 日志文件过期时长
	flock    sync.RWMutex // 同步锁
	nameLink string       // 写入用日志文件名
	nameOld  string       // 旧日志文件名
	nameNow  string       // 当前日志文件名
	pathLink string       // 写入用日志路径
	pathNow  string       // 当前日志路径
	logDir   string       // 日志文件夹
	logLevel int          // 日志等级
	maxDays  int          // 文件有效时间
	out      io.Writer    // io写入
	err      error        // 错误信息
	enablegz bool         // 是否允许gzip压缩旧日志文件
	debug    bool         // 是否调试模式
}

// LoggerWithRolling 滚动日志
func LoggerWithRolling(logdir, filename string, maxdays, loglevel int, enablegz, debug bool) gin.HandlerFunc {
	t := time.Now()
	// 初始化
	f := &ginLogger{
		logDir:   logdir,
		logLevel: loglevel,
		// flock:    new(sync.Mutex),
		fname:    filename,
		fexpired: int64(maxdays) * 24 * 60 * 60,
		maxDays:  maxdays,
		nameLink: fmt.Sprintf("%s.current.log", filename),
		nameNow:  fmt.Sprintf("%s.%v.log", filename, t.Format(gopsu.FileTimeFromat)),
		pathLink: filepath.Join(logdir, fmt.Sprintf("%s.current.log", filename)),
		pathNow:  filepath.Join(logdir, fmt.Sprintf("%s.%v.log", filename, t.Format(gopsu.FileTimeFromat))),
		enablegz: enablegz,
		debug:    debug,
	}
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
		path := c.Request.URL.Path
		// raw := c.Request.URL.RawQuery

		c.Next()

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
		// if raw != "" {
		// 	path = path + "?" + raw
		// }
		if len(c.Params) > 0 {
			var raw = url.Values{}
			for _, v := range c.Params {
				raw.Add(v.Key, v.Value)
			}
			path += "?" + raw.Encode()
		}
		param.Path = path
		if len(param.Keys) == 0 {
			fmt.Fprint(gin.DefaultWriter, fmt.Sprintf("%v |%3d| %-10s | %-15s|%-4s %s\n%s",
				param.TimeStamp.Format(gopsu.LogTimeFormat),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
				param.ErrorMessage,
			))
		} else {
			jsn, _ := json.Marshal(param.Keys)
			fmt.Fprint(gin.DefaultWriter, fmt.Sprintf("%v |%3d| %-10s | %-15s|%-4s %s|%s\n%s",
				param.TimeStamp.Format(gopsu.LogTimeFormat),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
				jsn,
				param.ErrorMessage,
			))
		}
	}
}

// 按日期切分文件
func (f *ginLogger) rollingFile() bool {
	f.flock.Lock()
	defer f.flock.Unlock()

	t := time.Now()
	f.nameNow = fmt.Sprintf("%s.%v.log", f.fname, t.Format(gopsu.FileTimeFromat))
	// 比对文件名，若不同则重新设置io
	if f.nameNow == f.nameOld {
		return false
	}

	f.pathNow = filepath.Join(f.logDir, f.nameNow)
	// 关闭旧fno
	f.fno.Close()
	// 创建新日志
	f.newFile()
	// 清理旧日志
	f.clearFile()

	return true
}

// 压缩旧日志
func (f *ginLogger) zipFile(s string) {
	if !f.enablegz || len(s) == 0 || !gopsu.IsExist(filepath.Join(f.logDir, s)) {
		return
	}
	go func() {
		zfile := filepath.Join(f.logDir, s+".zip")
		ofile := filepath.Join(f.logDir, s)

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
		header.Name = s
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return
		}
		if _, err = io.Copy(writer, zipfile); err != nil {
			return
		}
		// 删除已压缩的旧日志
		os.Remove(filepath.Join(f.logDir, s))
	}()
}

// 清理旧日志
func (f *ginLogger) clearFile() {
	// 若未设置超时，则不清理
	if f.fexpired == 0 {
		return
	}
	// 遍历文件夹
	lstfno, ex := ioutil.ReadDir(f.logDir)
	if ex != nil {
		ioutil.WriteFile("ginlogerr.log", []byte(fmt.Sprintf("clear log files error: %s", ex.Error())), 0644)
	}
	t := time.Now()
	for _, fno := range lstfno {
		if fno.IsDir() || !strings.Contains(fno.Name(), f.fname) || strings.Contains(fno.Name(), ".current") { // 忽略目录，不含日志名的文件，以及当前文件
			continue
		}
		// 比对文件生存期
		if t.Unix()-fno.ModTime().Unix() > f.fexpired {
			os.Remove(filepath.Join(f.logDir, fno.Name()))
		}
	}
}

// 创建新日志文件
func (f *ginLogger) newFile() {
	// 使用文件链接创建当前日志文件
	// 文件不存在时创建
	// if !gopsu.IsExist(f.pathNow) {
	// 	f, err := os.Create(f.pathNow)
	// 	if err == nil {
	// 		f.Close()
	// 	}
	// }
	// 删除旧的文件链接
	// os.Remove(f.pathLink)
	// // 创建当前日志链接
	// f.err = os.Symlink(f.nameNow, f.pathLink)
	// if f.err != nil {
	// 	println("Symlink log file error: " + f.err.Error())
	// }

	// 直接写入当日日志
	f.pathLink = f.pathNow
	if f.fname == "" {
		f.out = os.Stdout
	} else {
		// 打开文件
		f.fno, f.err = os.OpenFile(f.pathLink, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
		if f.err != nil {
			ioutil.WriteFile("ginlogerr.log", []byte("Log file open error: "+f.err.Error()), 0644)
			f.out = io.MultiWriter(os.Stdout)
		} else {
			if f.logLevel <= 10 {
				f.out = io.MultiWriter(f.fno, os.Stdout)
			} else {
				f.out = io.MultiWriter(f.fno)
			}
			// 判断是否压缩旧日志
			if f.enablegz {
				f.zipFile(f.nameOld)
			}
		}
	}
	f.nameOld = f.nameNow
}
