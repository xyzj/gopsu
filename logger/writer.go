/*
Package logger 日志专用写入器，可设置是否自动依据日期以及文件大小滚动日志文件
*/
package logger

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/loopfunc"
	"github.com/xyzj/gopsu/pathtool"
)

const (
	fileTimeFormat = "060102"   // 日志文件命名格式
	maxFileSize    = 1048576000 // 1G
	// ShortTimeFormat 日志事件戳格式
	ShortTimeFormat = "15:04:05.000 "
)

var (
	lineEnd = []byte{10}
)

// OptLog OptLog
type OptLog struct {
	// Filename 日志文件名，不需要扩展名，会自动追加时间戳以及.log扩展名，为空时其他参数无效，仅输出到控制台
	Filename string
	// FileDir 日志存放目录
	FileDir string
	// AutoRoll 是否自动滚动日志文件，true-依据MaxDays和MaxSize自动切分日志文件，日志文件名会额外追加日期时间戳‘yymmdd’和序号
	AutoRoll bool
	// MaxDays 日志最大保留天数，AutoRoll==true时有效
	MaxDays int
	// MaxSize 单个日志文件最大大小，AutoRoll==true时有效
	MaxSize int64
	// ZipFile 是否压缩旧日志文件，AutoRoll==true时有效
	ZipFile bool
	// SyncToConsole 同步输出到控制台
	SyncToConsole bool
	// DelayWrite 延迟写入，每秒检查写入缓存，并写入文件，非实时写入
	DelayWrite bool
}

// NewWriter 一个新的log写入器
//
// opt: 日志写入器配置
func NewWriter(opt *OptLog) io.Writer {
	if opt == nil {
		opt = &OptLog{}
	}
	if opt.Filename == "" {
		opt.AutoRoll = false
		opt.SyncToConsole = true
	}
	if opt.AutoRoll { // 检查关联参数
		if opt.MaxDays < 1 {
			opt.MaxDays = 1
		}
		if opt.MaxSize > 0 && opt.MaxSize < 10485760 {
			opt.MaxSize = 10485760
		}
		if opt.MaxSize == 0 {
			opt.MaxSize = maxFileSize
		}
	}
	t := time.Now()
	mylog := &Writer{
		cno:         os.Stdout,
		expired:     int64(opt.MaxDays)*24*60*60 - 10,
		fileMaxSize: opt.MaxSize,
		fname:       opt.Filename,
		rollfile:    opt.AutoRoll,
		fileDay:     t.Day(),
		fileHour:    t.Hour(),
		logDir:      opt.FileDir,
		chanGoWrite: make(chan []byte, 100),
		enablegz:    opt.ZipFile,
		withConsole: opt.SyncToConsole,
		withFile:    true,
		delayWrite:  opt.DelayWrite,
	}
	if opt.Filename != "" && opt.AutoRoll {
		ymd := t.Format(fileTimeFormat)
		for i := 1; i < 255; i++ {
			if !pathtool.IsExist(filepath.Join(mylog.logDir, fmt.Sprintf("%s.%s.%d.log", mylog.fname, ymd, i))) {
				mylog.fileIndex = byte(i) - 1
				break
			}
		}
	}
	mylog.newFile()
	mylog.startWrite()
	return mylog
}

// Writer 自定义Writer
type Writer struct {
	chanGoWrite  chan []byte
	chanEndWrite chan int
	cno          io.Writer
	fno          *os.File
	pathNow      string
	fname        string
	nameNow      string
	nameOld      string
	logDir       string
	expired      int64
	fileMaxSize  int64
	fileDay      int
	fileHour     int
	fileIndex    byte
	enablegz     bool
	rollfile     bool
	withConsole  bool
	withFile     bool
	delayWrite   bool
}

func (w *Writer) startWrite() {
	go loopfunc.LoopFunc(func(params ...interface{}) {
		tc := time.NewTicker(time.Minute * 10)
		tw := time.NewTicker(time.Second)
		buf := &bytes.Buffer{}
		buftick := &bytes.Buffer{}
		if !w.delayWrite {
			tw.Stop()
		}
		for {
			select {
			case p := <-w.chanGoWrite:
				buf.Reset()
				buf.Write(gopsu.Bytes(time.Now().Format(ShortTimeFormat)))
				buf.Write(p)
				if !bytes.HasSuffix(p, lineEnd) {
					buf.WriteByte(10)
				}
				if w.withFile {
					if w.delayWrite {
						buftick.Write(buf.Bytes())
					} else {
						w.fno.Write(buf.Bytes())
					}
				}
				if w.withConsole {
					w.cno.Write(buf.Bytes())
				}
				// w.out.Write(buf.Bytes())
			case <-tc.C:
				if w.rollfile {
					w.rollingFileNoLock()
				}
			case <-tw.C:
				if w.withFile {
					if buftick.Len() > 0 {
						w.fno.Write(buftick.Bytes())
						buftick.Reset()
					}
				}
			}
		}
	}, "log writer", nil)
}

// 创建新日志文件
func (w *Writer) newFile() {
	if w.fname == "" {
		w.withFile = false
		return
	}
	if w.rollfile {
		t := time.Now()
		if w.fileDay != t.Day() {
			w.fileDay = t.Day()
			w.fileIndex = 0
		}
		w.nameNow = fmt.Sprintf("%s.%s.%d.log", w.fname, t.Format(fileTimeFormat), w.fileIndex)
	} else {
		w.nameNow = fmt.Sprintf("%s.log", w.fname)
	}
	if w.nameOld == w.nameNow {
		return
	}
	// 关闭旧fno
	if w.fno != nil {
		w.fno.Close()
	}
	w.pathNow = filepath.Join(w.logDir, w.nameNow)
	// 直接写入当日日志
	var err error
	w.fno, err = os.OpenFile(w.pathNow, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
	if err != nil {
		os.WriteFile("logerr.log", gopsu.Bytes("log file open error: "+err.Error()), 0664)
		w.withFile = false
		return
	}
	w.withFile = true
	// if w.withConsole {
	// 	w.out = io.MultiWriter(os.Stdout, w.fno)
	// } else {
	// w.out = w.fno
	// }
	// 判断是否压缩旧日志
	if w.enablegz {
		w.zipFile(w.nameOld)
	}
	w.fno.Write(lineEnd)
}

// Write 异步写入日志，返回固定为 0, nil
func (w *Writer) Write(p []byte) (n int, err error) {
	// w.buf.Reset()
	// w.buf.Write(gopsu.Bytes(time.Now().Format(ShortTimeFormat)))
	// w.buf.Write(p)
	// if p[len(p)-1] != 10 {
	// 	w.buf.WriteByte(10)
	// }
	w.chanGoWrite <- p
	return 0, nil
}

// 检查文件大小,返回是否需要切分文件
func (w *Writer) rolledWithFileSize() bool {
	if w.fileHour == time.Now().Hour() {
		return false
	}
	w.fileHour = time.Now().Hour()
	fs, ex := os.Stat(w.pathNow)
	if ex == nil {
		if fs.Size() > w.fileMaxSize {
			if w.fileIndex >= 255 {
				w.fileIndex = 0
			} else {
				w.fileIndex++
			}
			return true
		}
	}
	return false
}

func (w *Writer) rollingFileNoLock() bool {
	if time.Now().Day() == w.fileDay && !w.rolledWithFileSize() {
		return false
	}
	// t := time.Now()
	// w.nameNow = fmt.Sprintf("%s.%v.%d.log", w.fname, t.Format(fileTimeFormat), w.fileIndex)
	// // 比对文件名，若不同则重新设置io
	// if w.nameNow == w.nameOld {
	// 	return false
	// }
	// 创建新日志
	w.newFile()
	// 清理旧日志
	w.clearFile()

	return true
}

// 压缩旧日志
func (w *Writer) zipFile(s string) {
	if !w.enablegz || len(s) == 0 || !pathtool.IsExist(filepath.Join(w.logDir, s)) {
		return
	}
	go func(s string) {
		err := zipFile(w.logDir, s, true)
		if err != nil {
			println("zip log file error: " + s + " " + err.Error())
			return
		}
	}(s)
}

// 清理旧日志
func (w *Writer) clearFile() {
	// 若未设置超时，则不清理
	if !w.rollfile || w.expired <= 0 {
		return
	}
	go func() {
		defer func() { recover() }()
		// 遍历文件夹
		lstfno, ex := os.ReadDir(w.logDir)
		if ex != nil {
			println(fmt.Sprintf("clear log files error: %s", ex.Error()))
			return
		}
		t := time.Now()
		for _, d := range lstfno {
			if d.IsDir() { // 忽略目录，不含日志名的文件，以及当前文件
				continue
			}
			fno, err := d.Info()
			if err != nil {
				continue
			}
			if !strings.Contains(fno.Name(), w.fname) {
				continue
			}
			// 比对文件生存期
			if t.Unix()-fno.ModTime().Unix() >= w.expired {
				os.Remove(filepath.Join(w.logDir, fno.Name()))
			}
		}
	}()
}

func zipFile(d, s string, delold bool) error {
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
