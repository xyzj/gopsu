package ginmiddleware

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-contrib/cors"
	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/unrolled/secure"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
	json "github.com/xyzj/gopsu/json"
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
	// 黑名单
	r.Use(Blacklist(""))
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
	r.Static("/static", gopsu.JoinPathFromHere("static"))
	return r
}

// GetSocketTimeout 获取超时时间
func GetSocketTimeout() time.Duration {
	return getSocketTimeout()
}
func getSocketTimeout() time.Duration {
	var t = 120
	b, err := ioutil.ReadFile(".sockettimeout")
	if err == nil {
		t = gopsu.String2Int(gopsu.TrimString(gopsu.String(b)), 10)
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
	return ListenAndServeTLS(port, h, "", "")
}

// ListenAndServeTLS 启用TLS监听
// port：端口号
// h： http.hander, like gin.New()
// certfile： cert file path
// keyfile： key file path
// clientca: 客户端根证书用于验证客户端合法性
func ListenAndServeTLS(port int, h *gin.Engine, certfile, keyfile string, clientca ...string) error {
	// 路由处理
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
	// 设置全局超时
	st := getSocketTimeout()
	// 初始化
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		ReadTimeout:  st,
		WriteTimeout: st,
		IdleTimeout:  st,
	}
	// 设置日志
	var writer io.Writer
	if gin.Mode() == gin.ReleaseMode {
		writer = io.MultiWriter(gin.DefaultWriter, os.Stdout)
	} else {
		writer = io.MultiWriter(gin.DefaultWriter)
	}
	// 启动http服务
	if strings.TrimSpace(certfile)+strings.TrimSpace(keyfile) == "" {
		fmt.Fprintf(writer, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Success start HTTP server at :"+strconv.Itoa(port))
		return s.ListenAndServe()
	}
	// 初始化证书
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
	s.TLSConfig = tc
	// 启动证书维护线程
	go renewCA(s, certfile, keyfile)
	// 启动https
	fmt.Fprintf(writer, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Success start HTTPS server at :"+strconv.Itoa(port))
	return s.ListenAndServeTLS("", "")
}

// 后台更新证书
func renewCA(s *http.Server, certfile, keyfile string) {
RUN:
	func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Fprintf(io.MultiWriter(gin.DefaultWriter, os.Stdout), "cert update crash: %s\n", err.(error).Error())
			}
		}()
		for range time.After(time.Hour * time.Duration(1+rand.Int31n(5))) {
			newcert, err := tls.LoadX509KeyPair(certfile, keyfile)
			if err == nil {
				s.TLSConfig.Certificates[0] = newcert
			}
		}
	}()
	time.Sleep(time.Second)
	goto RUN
}

// XForwardedIP 替换realip
func XForwardedIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, v := range []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
			if ip := c.Request.Header.Get(v); ip != "" {
				_, b, err := net.SplitHostPort(c.Request.RemoteAddr)
				if err != nil {
					c.Request.RemoteAddr = ip + ":" + b
				}
			}
		}
	}
}

// CFConnectingIP get cf ip
func CFConnectingIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if ip := c.Request.Header.Get("CF-Connecting-IP"); ip != "" {
			_, b, err := net.SplitHostPort(c.Request.RemoteAddr)
			if err != nil {
				c.Request.RemoteAddr = ip + ":" + b
			}
		}
	}
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
		var bodyjs string
		switch ct {
		case "", "application/x-www-form-urlencoded", "application/json":
			var x = url.Values{}
			// 先检查url参数
			x, _ = url.ParseQuery(c.Request.URL.RawQuery)
			// 检查body，若和url里面出现相同的关键字，以body内容为准
			if b, err := ioutil.ReadAll(c.Request.Body); err == nil {
				ans := gjson.ParseBytes(b)
				if ans.IsObject() { // body是json
					ans.ForEach(func(key gjson.Result, value gjson.Result) bool {
						x.Set(key.String(), value.String())
						return true
					})
					bodyjs = ans.String()
				} else { // body不是json，按urlencode处理
					if len(b)+len(c.Request.URL.RawQuery) > 0 {
						bodyjs = strings.Join([]string{c.Request.URL.RawQuery, gopsu.String(b)}, "&")
						xbody, _ := url.ParseQuery(gopsu.String(b))
						for k := range xbody {
							x.Set(k, xbody.Get(k))
						}
					}
				}
			}
			for k := range x {
				if strings.HasPrefix(k, "_") {
					continue
				}
				c.Params = append(c.Params, gin.Param{
					Key:   k,
					Value: x.Get(k),
				})
				// if k == "cachetag" || k == "cachestart" || k == "cacherows" {
				// 	continue
				// }
			}
			if len(bodyjs) > 0 {
				c.Params = append(c.Params, gin.Param{
					Key:   "_body",
					Value: bodyjs,
				})
			}
			return
		case "multipart/form-data":
			if mf, err := c.MultipartForm(); err == nil {
				if len(mf.Value) == 0 {
					return
				}
				if b, err := json.Marshal(mf.Value); err == nil {
					c.Params = append(c.Params, gin.Param{
						Key:   "_body",
						Value: gopsu.String(b),
					})
				}
				for k, v := range mf.Value {
					if strings.HasPrefix(k, "_") {
						continue
					}
					c.Params = append(c.Params, gin.Param{
						Key:   k,
						Value: strings.Join(v, ","),
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
			t, _ := strconv.Atoi(gopsu.TrimString(gopsu.String(b)))
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
//  r: 每秒可访问次数,1-100
//  b: 缓冲区大小
func RateLimit(r, b int) gin.HandlerFunc {
	if r < 1 || r > 100 {
		r = 5
	}
	var limiter = rate.NewLimiter(rate.Every(time.Millisecond*time.Duration(1000/r)), b)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}

// RateLimitWithIP ip限流器，基于官方库
//  r: 每秒可访问次数,1-100
//  b: 缓冲区大小
func RateLimitWithIP(r, b int) gin.HandlerFunc {
	if r < 1 || r > 100 {
		r = 5
	}
	var cliMap sync.Map
	return func(c *gin.Context) {
		limiter, _ := cliMap.LoadOrStore(c.ClientIP(), rate.NewLimiter(rate.Every(time.Millisecond*time.Duration(1000/r)), b))
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := limiter.(*rate.Limiter).WaitN(ctx, 1); err != nil {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		// if !limiter.(*rate.Limiter).Allow() {
		// 	c.AbortWithStatus(http.StatusTooManyRequests)
		// 	return
		// }
		c.Next()
	}
}

// RateLimitWithTimeout 超时限流器，基于官方库
//  r: 每秒可访问次数,1-100
//  b: 缓冲区大小
//  t: 超时时长
func RateLimitWithTimeout(r, b int, t time.Duration) gin.HandlerFunc {
	if r < 1 || r > 100 {
		r = 5
	}
	var limiter = rate.NewLimiter(rate.Every(time.Millisecond*time.Duration(1000/r)), b)
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), t)
		defer cancel()
		if err := limiter.WaitN(ctx, 1); err != nil {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
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
					var s string
					if b, err := json.Marshal(ans); err != nil {
						s = gopsu.String(b)
					}
					// s, _ := json.MarshalToString(ans)
					c.Params = append(c.Params, gin.Param{
						Key:   "_cacheData",
						Value: s,
					})
				}
			}
		}
	}
}

type abortRecord struct {
	locker sync.RWMutex
	abortC map[string]uint64
}

func (ar *abortRecord) Add(ip string) {
	ar.locker.Lock()
	defer ar.locker.Unlock()
	if v, ok := ar.abortC[ip]; !ok {
		ar.abortC[ip] = 1
	} else {
		atomic.AddUint64(&v, 1)
	}
}

// Blacklist IP黑名单
func Blacklist(filename string) gin.HandlerFunc {
	if filename == "" {
		filename = gopsu.JoinPathFromHere(".blacklist")
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		b = []byte{}
	}
	ips := make([]string, 0)
	chkblack := false
	for _, v := range strings.Split(string(b), "\n") {
		if net.ParseIP(v) != nil {
			ips = append(ips, v)
		}
	}
	if len(ips) > 0 {
		chkblack = true
	}
	return func(c *gin.Context) {
		if !chkblack {
			return
		}

		ip := c.ClientIP()
		for _, v := range ips {
			if v == ip {
				c.AbortWithStatus(418)
				return
			}
		}
	}
}

// AntiDDOS 阻止超流量ip
func AntiDDOS() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}
