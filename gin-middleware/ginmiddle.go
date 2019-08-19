package ginmiddleware

import (
	"crypto/tls"
	"crypto/x509"
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
	"github.com/xyzj/gopsu/db"
)

// NewGinEngine 返回一个新的gin路由
// logName：日志文件名
// logDays：日志保留天数
// logLevel：日志等级
// logGZ：是否压缩归档日志
// debug：是否使用调试模式
func NewGinEngine(logDir, logName string, logDays, logLevel int) *gin.Engine {
	r := gin.New()
	// 中间件
	// 日志
	r.Use(LoggerWithRolling(logDir, logName, logDays, logLevel))
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
func ListenAndServe(port int, h *gin.Engine) error {
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
	fmt.Fprintf(gin.DefaultWriter, "%s [90] [%s] %s\n", time.Now().Format(gopsu.LogTimeFormat), "HTTP", "Success start HTTP server at :"+strconv.Itoa(port))
	return s.ListenAndServe()
}

// ListenAndServeTLS 启用TLS监听
// port：端口号
// timeout：读写超时
// h： http.hander, like gin.New()
// certfile： cert file path
// keyfile： key file path
// clientca: 客户端根证书用于验证客户端合法性
func ListenAndServeTLS(port int, h *gin.Engine, certfile, keyfile string, clientca ...string) error {
	if gopsu.IsExist(".forcehttp") {
		return ListenAndServe(port, h)
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
	var tc = &tls.Config{}
	if len(clientca) > 0 {
		pool := x509.NewCertPool()
		caCrt, err := ioutil.ReadFile(clientca[0])
		if err == nil {
			pool.AppendCertsFromPEM(caCrt)
			tc.ClientCAs = pool
			tc.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		ReadTimeout:  getSocketTimeout(),
		WriteTimeout: getSocketTimeout(),
		IdleTimeout:  getSocketTimeout(),
		TLSConfig:    tc,
	}

	fmt.Fprintf(gin.DefaultWriter, "%s [90] [%s] %s\n", time.Now().Format(gopsu.LogTimeFormat), "HTTP", "Success start HTTPS server at :"+strconv.Itoa(port))
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

// ReadCacheJSON 读取数据库缓存
func ReadCacheJSON(mydb *db.MySQL) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mydb != nil {
			cachetag := c.Param("cachetag")
			if cachetag != "" {
				if gopsu.IsExist(cachetag) {
					cachestart := gopsu.String2Int(c.Param("cachestart"), 10)
					cacherows := gopsu.String2Int(c.Param("cachesrows"), 10)
					ans := mydb.QueryCacheJSON(cachetag, cachestart, cacherows)
					if gjson.Parse(ans).Get("total").Int() > 0 {
						c.Params = append(c.Params, gin.Param{
							Key:   "cacheData",
							Value: ans,
						})
					}
				}
			}
		}
		c.Next()
	}
}

// ReadCachePB2 读取数据库缓存
func ReadCachePB2(mydb *db.MySQL) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mydb != nil {
			cachetag := c.Param("cachetag")
			if cachetag != "" {
				if gopsu.IsExist(cachetag) {
					cachestart := gopsu.String2Int(c.Param("cachestart"), 10)
					cacherows := gopsu.String2Int(c.Param("cachesrows"), 10)
					ans := mydb.QueryCachePB2(cachetag, cachestart, cacherows)
					if ans.Total > 0 {
						b, _ := ans.Marshal()
						c.Params = append(c.Params, gin.Param{
							Key:   "cacheData",
							Value: string(b),
						})
					}
				}
			}
		}
		c.Next()
	}
}
