package ginmiddleware

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/unrolled/secure"
	"github.com/xyzj/gopsu"
	db "github.com/xyzj/gopsu/db"
	"github.com/xyzj/gopsu/rate"
)

// NewGinEngine 返回一个新的gin路由
// logName：日志文件名
// logDays：日志保留天数
// logLevel：日志等级（已废弃）
func NewGinEngine(logDir, logName string, logDays int, logLevel ...int) *gin.Engine {
	r := gin.New()
	// 中间件
	//cors
	r.Use(cors.New(cors.Config{
		MaxAge:           time.Hour * 24,
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowWildcard:    true,
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
	}))
	// 日志
	r.Use(LoggerWithRolling(logDir, logName, logDays))
	// 错误恢复
	// r.Use(gin.Recovery())
	r.Use(Recovery())
	// 数据压缩
	r.Use(gingzip.Gzip(9))
	// 读取请求参数
	// r.Use(ReadParams())
	// 渲染模板
	// r.HTMLRender = multiRender()
	// 基础路由
	// 404,405
	r.HandleMethodNotAllowed = true
	r.NoMethod(Page405)
	r.NoRoute(Page404)
	// r.GET("/", PageDefault)
	// r.POST("/", PageDefault)
	r.GET("/health", PageDefault)
	r.GET("/clearlog", CheckRequired("name"), Clearlog)
	r.GET("/runtime", PageRuntime)
	r.Static("/static", gopsu.JoinPathFromHere("static"))
	return r
}

func getSocketTimeout() time.Duration {
	var t = 120
	b, err := ioutil.ReadFile(".sockettimeout")
	if err == nil {
		t = gopsu.String2Int(gopsu.TrimString(string(b)), 10)
	}
	if t < 120 {
		t = 120
	}
	return time.Second * time.Duration(t)
}

// 遍历所有路由
// func getRoutes(h *gin.Engine) string {
// 	var sss string
// 	for _, v := range h.Routes() {
// 		if strings.ContainsAny(v.Path, "*") && !strings.HasSuffix(v.Path, "filepath") {
// 			return ""
// 		}
// 		if v.Path == "/" || v.Method == "HEAD" || strings.HasSuffix(v.Path, "*filepath") {
// 			continue
// 		}
// 		sss += fmt.Sprintf(`<a>%s: %s</a><br><br>`, v.Method, v.Path)
// 	}
// 	return sss
// }

// ListenAndServe 启用监听
// port：端口号
// h： http.hander, like gin.New()
func ListenAndServe(port int, h *gin.Engine) error {
	st := getSocketTimeout()
	var findRoot = false
	for _, v := range h.Routes() {
		if v.Path == "/" {
			findRoot = true
			break
		}
	}
	if !findRoot {
		h.GET("/", PageDefault)
	}
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		ReadTimeout:  st,
		WriteTimeout: st,
		IdleTimeout:  st,
	}
	var writer io.Writer
	if gin.Mode() == gin.ReleaseMode {
		writer = io.MultiWriter(gin.DefaultWriter, os.Stdout)
	} else {
		writer = io.MultiWriter(gin.DefaultWriter)
	}
	fmt.Fprintf(writer, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Success start HTTP server at :"+strconv.Itoa(port))
	return s.ListenAndServe()
}

// ListenAndServeTLS 启用TLS监听
// port：端口号
// h： http.hander, like gin.New()
// certfile： cert file path
// keyfile： key file path
// clientca: 客户端根证书用于验证客户端合法性
func ListenAndServeTLS(port int, h *gin.Engine, certfile, keyfile string, clientca ...string) error {
	var tc = &tls.Config{
		Certificates: make([]tls.Certificate, 1),
	}
	var err error
	tc.Certificates[0], err = tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		return err
	}
	if len(clientca) > 0 {
		pool := x509.NewCertPool()
		caCrt, err := ioutil.ReadFile(clientca[0])
		if err == nil {
			pool.AppendCertsFromPEM(caCrt)
			tc.ClientCAs = pool
			tc.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}
	var findRoot = false
	for _, v := range h.Routes() {
		if v.Path == "/" {
			findRoot = true
			break
		}
	}
	if !findRoot {
		h.GET("/", PageDefault)
	}
	st := getSocketTimeout()
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		ReadTimeout:  st,
		WriteTimeout: st,
		IdleTimeout:  st,
		TLSConfig:    tc,
	}
	go func() {
		var runLook sync.WaitGroup
	RUN:
		runLook.Add(1)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					fmt.Fprintf(io.MultiWriter(gin.DefaultWriter, os.Stdout), "cert update crash: %s\n", err.(error).Error())
				}
				runLook.Done()
			}()
			tt := time.NewTicker(time.Hour * 24)
			for range tt.C {
				newcert, err := tls.LoadX509KeyPair(certfile, keyfile)
				if err == nil {
					s.TLSConfig.Certificates[0] = newcert
				}
			}
		}()
		time.Sleep(time.Second)
		runLook.Wait()
		goto RUN
	}()
	var writer io.Writer
	if gin.Mode() == gin.ReleaseMode {
		writer = io.MultiWriter(gin.DefaultWriter, os.Stdout)
	} else {
		writer = io.MultiWriter(gin.DefaultWriter)
	}
	fmt.Fprintf(writer, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Success start HTTPS server at :"+strconv.Itoa(port))
	return s.ListenAndServeTLS("", "")
}

// CheckRequired 检查必填参数
func CheckRequired(params ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, v := range params {
			if gopsu.TrimString(v) == "" {
				continue
			}
			if c.Param(v) == "" {
				c.Set("status", 0)
				c.Set("detail", v)
				c.Set("xfile", 5)
				js, _ := sjson.Set("", "key_name", v)
				c.Set("xfile_args", gjson.Parse(js).Value())
				c.AbortWithStatusJSON(http.StatusBadRequest, c.Keys)
				break
			}
		}
	}
}

// HideParams 隐藏敏感参数值
func HideParams(params ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if gin.IsDebugging() {
			return
		}
		replaceP := make([]string, 0)
		body := c.Params.ByName("_body")
		jsbody := gjson.Parse(body).Exists()
		// 创建url替换器，并替换_body
		for _, v := range params {
			replaceP = append(replaceP, v+"="+c.Params.ByName(v))
			replaceP = append(replaceP, v+"=**classified**")
			if len(body) > 0 && jsbody {
				body, _ = sjson.Set(body, v, "**classified**")
			}
		}
		r := strings.NewReplacer(replaceP...)
		c.Request.RequestURI = r.Replace(c.Request.RequestURI)
		if !jsbody { // 非json body尝试替换字符串
			body = r.Replace(body)
		}
		for k, v := range c.Params {
			if v.Key == "_body" {
				c.Params[k] = gin.Param{
					Key:   "_body",
					Value: body,
				}
				break
			}
		}
		c.Next()
	}
}

// ReadParams 读取请求的参数，保存到c.Params
func ReadParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ct = strings.Split(c.GetHeader("Content-Type"), ";")[0]
		var x = url.Values{}
		switch ct {
		case "", "application/x-www-form-urlencoded", "application/json", "multipart/form-data":
			x, _ = url.ParseQuery(c.Request.URL.RawQuery)
			if ct == "multipart/form-data" {
				break
			}
			b, err := ioutil.ReadAll(c.Request.Body)
			if err == nil {
				if len(b) > 0 {
					c.Params = append(c.Params, gin.Param{
						Key:   "_body",
						Value: string(b),
					})
					ans := gjson.ParseBytes(b)
					if ans.IsObject() {
						// if ct == "application/json" {
						// 	gjson.ParseBytes(b).ForEach(func(key, value gjson.Result) bool {
						ans.ForEach(func(key, value gjson.Result) bool {
							x.Add(key.String(), value.String())
							return true
						})
					} else {
						xx, _ := url.ParseQuery(string(b))
						for k, v := range xx {
							x.Add(k, v[0])
						}
					}
				}
			}
		}

		if len(x.Encode()) > 0 {
			for k := range x {
				if strings.HasPrefix(k, "_") {
					continue
				}
				c.Params = append(c.Params, gin.Param{
					Key:   k,
					Value: x.Get(k),
				})
			}
		}
	}
}

// ReadCacheJSON 读取数据库缓存
func ReadCacheJSON(mydb db.SQLInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mydb != nil {
			cachetag := c.Param("cachetag")
			if cachetag != "" {
				cachestart := gopsu.String2Int(c.Param("cachestart"), 10)
				cacherows := gopsu.String2Int(c.Param("cacherows"), 10)
				ans := mydb.QueryCacheJSON(cachetag, cachestart, cacherows)
				if gjson.Parse(ans).Get("total").Int() > 0 {
					c.Params = append(c.Params, gin.Param{
						Key:   "_cacheData",
						Value: ans,
					})
				}
			}
		}
	}
}

// ReadCachePB2 读取数据库缓存
func ReadCachePB2(mydb db.SQLInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		if mydb != nil {
			cachetag := c.Param("cachetag")
			if cachetag != "" {
				cachestart := gopsu.String2Int(c.Param("cachestart"), 10)
				cacherows := gopsu.String2Int(c.Param("cacherows"), 10)
				ans := mydb.QueryCachePB2(cachetag, cachestart, cacherows)
				if ans.Total > 0 {
					s, _ := json.MarshalToString(ans)
					c.Params = append(c.Params, gin.Param{
						Key:   "_cacheData",
						Value: s,
					})
				}
			}
		}
	}
}

// CheckSecurityCode 校验安全码
// codeType: 安全码更新周期，h: 每小时更新，m: 每分钟更新
// codeRange: 安全码容错范围（分钟）
func CheckSecurityCode(codeType string, codeRange int) gin.HandlerFunc {
	return func(c *gin.Context) {
		sc := c.GetHeader("Legal-High")
		found := false
		if len(sc) == 32 {
			for _, v := range gopsu.CalculateSecurityCode(codeType, time.Now().Month().String(), codeRange) {
				if v == sc {
					found = true
					break
				}
			}
		}
		if !found {
			c.Set("status", 0)
			c.Set("detail", "Illegal Security-Code")
			c.Set("xfile", 10)
			c.AbortWithStatusJSON(http.StatusUnauthorized, c.Keys)
		}
	}
}

// Delay 性能延迟
func Delay() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		b, err := ioutil.ReadFile(".performance")
		if err == nil {
			t, _ := strconv.Atoi(gopsu.TrimString(string(b)))
			if t > 5000 || t < 0 {
				t = 5000
			}
			time.Sleep(time.Millisecond * time.Duration(t))
		}
	}
}

// TLSRedirect tls重定向
func TLSRedirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		secureMiddleware := secure.New(secure.Options{
			SSLRedirect: true,
		})

		err := secureMiddleware.Process(c.Writer, c.Request)
		if err != nil {
			return
		}

		c.Next()
	}
}

// RateLimit 限流器，基于官方库
// 当前设定最小单位毫秒
func RateLimit(r, b int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Every(time.Millisecond*time.Duration(1000/r)), b)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}
