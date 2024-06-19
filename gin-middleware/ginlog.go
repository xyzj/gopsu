package ginmiddleware

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xyzj/gopsu"
	json "github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
)

type logParam struct {
	timer      time.Duration
	keys       map[string]any
	jsn        []byte
	clientIP   string
	method     string
	path       string
	token      string
	body       string
	username   string
	statusCode int
}

// LogToWriter LogToWriter
func LogToWriter(w io.Writer, skippath ...string) gin.HandlerFunc {
	// 设置io
	gin.DefaultWriter = w
	gin.DefaultErrorWriter = w
	chanlog := make(chan *logParam, 200)
	if len(skippath) == 0 {
		skippath = []string{"/showroutes", "/static"}
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		for a := range chanlog {
			if len(a.keys) > 0 {
				a.jsn, _ = json.Marshal(a.keys)
			}
			if a.token != "" {
				if a.username != "" {
					a.path = "(" + a.username + "-" + gopsu.CalcCRC32String(gopsu.Bytes(a.token)) + ")" + a.path
				} else {
					a.path = "(" + gopsu.CalcCRC32String(gopsu.Bytes(a.token)) + ")" + a.path
				}
			}
			if a.body != "" {
				a.path += " |" + a.body
			}
			s := fmt.Sprintf("|%3d |%-13s|%-15s|%-4s %s |%s", a.statusCode, a.timer, a.clientIP, a.method, a.path, a.jsn)
			w.Write(json.Bytes(s))
			if gin.IsDebugging() {
				println(time.Now().Format(logger.ShortTimeFormat) + s)
			}
		}
	}, "http log", w)
	return func(c *gin.Context) {
		// |,(,) 124,40,41,32
		for _, v := range skippath {
			if strings.HasPrefix(c.Request.URL.Path, v) {
				return
			}
		}
		start := time.Now()
		c.Next()
		// Stop timer
		chanlog <- &logParam{
			timer:      time.Since(start),
			path:       c.Request.URL.Path,
			token:      c.GetHeader("User-Token"),
			body:       c.Param("_body"),
			clientIP:   c.ClientIP(),
			method:     c.Request.Method,
			statusCode: c.Writer.Status(),
			username:   c.Param("_userTokenName"),
			keys:       c.Keys,
		}
	}
}

// var md5worker = gopsu.GetNewCryptoWorker(gopsu.CryptoMD5)

// type ginLogger struct {
// 	fno       *os.File     // 文件日志
// 	fname     string       // 日志文件名
// 	fileIndex byte         // 文件索引号
// 	expired   int64        // 日志文件过期时长
// 	flock     sync.RWMutex // 同步锁
// 	nameOld   string       // 旧日志文件名
// 	nameNow   string       // 当前日志文件名
// 	fileHour  int          // 旧时间戳
// 	fileDay   int          // 日期戳
// 	pathOld   string       // 写入用日志路径
// 	pathNow   string       // 当前日志路径
// 	logDir    string       // 日志文件夹
// 	maxDays   int          // 文件有效时间
// 	out       io.Writer    // io写入
// 	err       error        // 错误信息
// 	enablegz  bool         // 是否允许gzip压缩旧日志文件
// 	skipPath  []string     // 不记录日志的路由
// }

// LoggerWithRolling 滚动日志
// logdir: 日志存放目录。
// filename：日志文件名。
// maxdays：日志文件最大保存天数。
func LoggerWithRolling(logdir, filename string, maxdays int, skippath ...string) gin.HandlerFunc {
	lo := logger.NewWriter(&logger.OptLog{
		AutoRoll:     true,
		FileDir:      logdir,
		Filename:     filename,
		MaxDays:      maxdays,
		CompressFile: true,
		DelayWrite:   true,
	})
	return LogToWriter(lo, skippath...)
	// return LoggerWithRollingSkip(logdir, filename, maxdays, []string{"/static"})
	// }
	// func LoggerWithRollingSkip(logdir, filename string, maxdays int, skippath []string) gin.HandlerFunc {
	// t := time.Now()
	// // 初始化
	// f := &ginLogger{
	// 	logDir:   logdir,
	// 	fname:    filename,
	// 	expired:  int64(maxdays)*24*60*60 - 10,
	// 	maxDays:  maxdays,
	// 	fileHour: t.Hour(),
	// 	fileDay:  t.Day(),
	// 	pathOld:  filepath.Join(logdir, fmt.Sprintf("%s.current.log", filename)),
	// 	enablegz: true,
	// 	skipPath: skippath,
	// }
	// if f.maxDays <= 1 {
	// 	f.fname = ""
	// }
	// // 搜索最后一个文件名
	// for i := byte(255); i > 0; i-- {
	// 	if pathtool.IsExist(filepath.Join(f.logDir, fmt.Sprintf("%s.%v.%d.log", filename, t.Format(gopsu.FileTimeFormat), i))) {
	// 		f.fileIndex = i
	// 	}
	// }
	// // 创建新日志
	// f.newFile()
	// // 设置io
	// gin.DefaultWriter = f.out
	// gin.DefaultErrorWriter = f.out
	// // 创建写入线程
	// var chanWriteLog = make(chan string, 100)
	// go func() {
	// 	tc := time.NewTicker(time.Minute * 10)
	// RUN:
	// 	func() {
	// 		defer func() {
	// 			recover()
	// 		}()
	// 		time.Sleep(time.Second * 3)
	// 		for {
	// 			select {
	// 			case s := <-chanWriteLog:
	// 				fmt.Fprintln(f.out, s)
	// 			case <-tc.C:
	// 				// 检查是否需要切分文件
	// 				if f.rollingFile() {
	// 					gin.DefaultWriter = f.out
	// 					gin.DefaultErrorWriter = f.out
	// 				}
	// 			}
	// 		}
	// 	}()
	// 	goto RUN
	// }()
	// return func(c *gin.Context) {
	// 	if f.maxDays <= 0 {
	// 		return
	// 	}
	// 	for _, v := range f.skipPath {
	// 		if strings.HasPrefix(c.Request.RequestURI, v) {
	// 			return
	// 		}
	// 	}
	// 	start := time.Now()
	// 	token := c.GetHeader("User-Token")
	// 	if len(token) == 36 {
	// 		token = md5worker.Hash(gopsu.Bytes(token))
	// 	}
	// 	c.Next()

	// 	path := c.Request.URL.Path
	// 	param := &gin.LogFormatterParams{
	// 		Request: c.Request,
	// 		Keys:    c.Keys,
	// 	}
	// 	// Stop timer
	// 	param.TimeStamp = time.Now()
	// 	param.Latency = time.Since(start)
	// 	param.ClientIP = c.ClientIP()
	// 	param.Method = c.Request.Method
	// 	param.StatusCode = c.Writer.Status()
	// 	// param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()
	// 	param.ErrorMessage = c.Errors.String()
	// 	param.BodySize = c.Writer.Size()
	// 	if body, ok := c.Params.Get("_body"); ok {
	// 		path += "|" + body
	// 	}
	// 	if len(token) == 32 {
	// 		path = "(" + token + ")" + path
	// 	}
	// 	param.Path = path

	// 	var s string
	// 	if len(param.Keys) == 0 {
	// 		s = fmt.Sprintf("%v |%3d| %-10s | %-15s|%-4s %s",
	// 			param.TimeStamp.Format(gopsu.ShortTimeFormat),
	// 			param.StatusCode,
	// 			param.Latency,
	// 			param.ClientIP,
	// 			param.Method,
	// 			param.Path,
	// 		)
	// 	} else {
	// 		jsn, _ := json.Marshal(param.Keys)
	// 		s = fmt.Sprintf("%v |%3d| %-10s | %-15s|%-4s %s ▸%s",
	// 			param.TimeStamp.Format(gopsu.ShortTimeFormat),
	// 			param.StatusCode,
	// 			param.Latency,
	// 			param.ClientIP,
	// 			param.Method,
	// 			param.Path,
	// 			jsn,
	// 		)
	// 	}
	// 	if param.ErrorMessage != "" {
	// 		s += " #" + param.ErrorMessage
	// 	}
	// 	chanWriteLog <- s
	// }
}

// // 检查文件大小,返回是否需要切分文件
// func (f *ginLogger) rolledWithFileSize() bool {
// 	if f.fileHour == time.Now().Hour() {
// 		return false
// 	}
// 	f.fileHour = time.Now().Hour()
// 	fs, ex := os.Stat(f.pathNow)
// 	if ex == nil {
// 		if fs.Size() > 1048576000 {
// 			if f.fileIndex >= 255 {
// 				f.fileIndex = 0
// 			} else {
// 				f.fileIndex++
// 			}
// 			return true
// 		}
// 	}
// 	return false
// }

// // 按日期切分文件
// func (f *ginLogger) rollingFile() bool {
// 	if f.fname == "" {
// 		return false
// 	}
// 	f.flock.Lock()
// 	defer f.flock.Unlock()

// 	t := time.Now()
// 	f.rolledWithFileSize()
// 	f.nameNow = fmt.Sprintf("%s.%v.%d.log", f.fname, t.Format(gopsu.FileTimeFormat), f.fileIndex)
// 	// 比对文件名，若不同则重新设置io
// 	if f.nameNow == f.nameOld {
// 		return false
// 	}
// 	// 创建新日志
// 	f.newFile()
// 	// 清理旧日志
// 	f.cleanFile()

// 	return true
// }

// // 压缩旧日志
// func (f *ginLogger) zipFile(s string) {
// 	if f.fname == "" || !f.enablegz || len(s) == 0 || !pathtool.IsExist(filepath.Join(f.logDir, s)) {
// 		return
// 	}
// 	go func(s string) {
// 		err := gopsu.ZIPFile(f.logDir, s, true)
// 		if err != nil {
// 			fmt.Fprintln(f.out, "zip log file error: "+s+" "+err.Error())
// 			return
// 		}
// 	}(s)
// }

// // 清理旧日志
// func (f *ginLogger) cleanFile() {
// 	// 若未设置超时，则不清理
// 	if f.fname == "" || f.expired == 0 {
// 		return
// 	}
// 	go func() {
// 		defer func() { recover() }()

// 		// 遍历文件夹
// 		lstfno, ex := ioutil.ReadDir(f.logDir)
// 		if ex != nil {
// 			os.WriteFile("ginlogerr.log", gopsu.Bytes(fmt.Sprintf("clear log files error: %s", ex.Error())), 0664)
// 			return
// 		}
// 		t := time.Now()
// 		for _, fno := range lstfno {
// 			if fno.IsDir() || !strings.Contains(fno.Name(), f.fname) { // 忽略目录，不含日志名的文件，以及当前文件
// 				continue
// 			}
// 			// 比对文件生存期
// 			if t.Unix()-fno.ModTime().Unix() >= f.expired {
// 				os.Remove(filepath.Join(f.logDir, fno.Name()))
// 			}
// 		}
// 	}()
// }

// // 创建新日志文件
// func (f *ginLogger) newFile() {
// 	if f.fname == "" {
// 		f.out = io.MultiWriter(os.Stdout)
// 		return
// 	}
// 	t := time.Now()
// 	if f.fileDay != t.Day() {
// 		f.fileDay = t.Day()
// 		f.fileIndex = 0
// 	}
// 	if f.fno != nil {
// 		f.fno.Close()
// 	}
// 	// 直接写入当日日志
// 	f.nameNow = fmt.Sprintf("%s.%v.%d.log", f.fname, t.Format(gopsu.FileTimeFormat), f.fileIndex)
// 	f.pathNow = filepath.Join(f.logDir, f.nameNow)
// 	f.pathOld = f.pathNow

// 	// 打开文件
// 	f.fno, f.err = os.OpenFile(f.pathOld, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0664)
// 	if f.err != nil {
// 		os.WriteFile("ginlogerr.log", gopsu.Bytes("Log file open error: "+f.err.Error()), 0664)
// 		f.out = io.MultiWriter(os.Stdout)
// 	} else {
// 		if gin.Mode() == "debug" {
// 			f.out = io.MultiWriter(f.fno, os.Stdout)
// 		} else {
// 			f.out = io.MultiWriter(f.fno)
// 		}
// 	}
// 	// 判断是否压缩旧日志
// 	if f.enablegz {
// 		f.zipFile(f.nameOld)
// 	}
// 	f.nameOld = f.nameNow
// }
