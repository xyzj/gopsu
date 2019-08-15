package ginmiddleware

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
)

const (
	// LogTimeFormat 日志内容时间戳格式
	LogTimeFormat = "2006/01/02 15:04:05.000"
)

// NewGinEngine 返回一个新的gin路由
// logName：日志文件名
// logDays：日志保留天数
// logLevel：日志等级
// logGZ：是否压缩归档日志
// debug：是否使用调试模式
func NewGinEngine(logDir, logName string, logDays, logLevel int, logGZ, debug bool) *gin.Engine {
	r := gin.New()
	// 中间件
	// 日志
	r.Use(LoggerWithRolling(logDir, logName, logDays, logLevel, logGZ, debug))
	// 错误恢复
	r.Use(gin.Recovery())
	// 读取请求参数
	r.Use(ReadParams())
	// 数据压缩
	r.Use(gingzip.Gzip(9))
	// 渲染模板
	// r.HTMLRender = multiRender()
	// 基础路由
	// 404,405
	r.NoMethod(Page405)
	r.NoRoute(Page404)
	r.GET("/", PageDefault)
	r.POST("/", PageDefault)
	r.GET("/health", PageDefault)

	return r
}

func getSocketTimeout() time.Duration {
	var t = 60
	if gopsu.IsExist(".sockettimeout") {
		b, err := ioutil.ReadFile(".sockettimeout")
		if err == nil {
			t = gopsu.String2Int(string(b), 10)
		}
	}
	if t < 60 {
		t = 60
	}
	return time.Second * time.Duration(t)
}

// ListenAndServe 启用监听
// port：端口号
// timeout：读写超时
// h： http.hander, like gin.New()
func ListenAndServe(port int, h *gin.Engine, startMsg ...string) error {
	var sss string
	for _, v := range h.Routes() {
		if v.Path == "/" || v.Method == "HEAD" || strings.ContainsAny(v.Path, "*") {
			continue
		}
		sss += fmt.Sprintf(`<a>%s: %s</a><br><br>`, v.Method, v.Path)
	}
	h.GET("/showroutes", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.Status(http.StatusOK)
		render.WriteString(c.Writer, sss, nil)
	})
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		ReadTimeout:  getSocketTimeout(),
		WriteTimeout: getSocketTimeout(),
		IdleTimeout:  getSocketTimeout(),
	}
	if len(startMsg) > 0 {
		fmt.Fprintf(gin.DefaultWriter, "%s [%s] %s\n", time.Now().Format(LogTimeFormat), "HTTP", strings.Join(startMsg, " "))
	} else {
		fmt.Fprintf(gin.DefaultWriter, "%s [%s] %s\n", time.Now().Format(LogTimeFormat), "HTTP", "Success start HTTP server at :"+strconv.Itoa(port))
	}
	return s.ListenAndServe()
}

// ListenAndServeTLS 启用TLS监听
// port：端口号
// timeout：读写超时
// h： http.hander, like gin.New()
// certfile： cert file path
// keyfile： key file path
func ListenAndServeTLS(port int, h *gin.Engine, certfile, keyfile string, startMsg ...string) error {
	if !gopsu.IsExist(certfile) || !gopsu.IsExist(keyfile) {
		return fmt.Errorf("no cert or key file found")
	}
	var sss string
	for _, v := range h.Routes() {
		if v.Path == "/" || v.Method == "HEAD" || strings.ContainsAny(v.Path, "*") {
			continue
		}
		sss += fmt.Sprintf(`<a>%s: %s</a><br><br>`, v.Method, v.Path)
	}
	h.GET("/showroutes", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.Status(http.StatusOK)
		render.WriteString(c.Writer, sss, nil)
	})
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		ReadTimeout:  getSocketTimeout(),
		WriteTimeout: getSocketTimeout(),
		IdleTimeout:  getSocketTimeout(),
	}
	if len(startMsg) > 0 {
		fmt.Fprintf(gin.DefaultWriter, "%s [%s] %s\n", time.Now().Format(LogTimeFormat), "HTTP", strings.Join(startMsg, " "))
	} else {
		fmt.Fprintf(gin.DefaultWriter, "%s [%s] %s\n", time.Now().Format(LogTimeFormat), "HTTP", "Success start HTTPS server at :"+strconv.Itoa(port))
	}
	return s.ListenAndServeTLS(certfile, keyfile)
}

// CheckRequired 检查必填参数
func CheckRequired(params ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, v := range params {
			if c.Param(v) == "" {
				c.Set("status", 0)
				c.Set("detail", "Incomplete parameters: "+v)
				c.AbortWithStatusJSON(200, c.Keys)
				return
			}
		}
		c.Next()
	}
}

// ReadParams 读取请求的参数，保存到c.Params
func ReadParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		b, _ := c.GetRawData()
		m := gjson.ParseBytes(b)
		if m.Exists() {
			m.ForEach(func(key, value gjson.Result) bool {
				c.Params = append(c.Params, gin.Param{
					Key:   key.String(),
					Value: value.String(),
				})
				return true
			})
		} else {
			var x url.Values
			switch c.Request.Method {
			case "GET":
				x, _ = url.ParseQuery(c.Request.URL.RawQuery)
			case "POST":
				x, _ = url.ParseQuery(string(b))
			}
			for k := range x {
				c.Params = append(c.Params, gin.Param{
					Key:   k,
					Value: x.Get(k),
				})
			}
		}
		c.Next()
	}
}
