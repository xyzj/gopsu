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

	"github.com/gin-contrib/cors"
	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/gogo/protobuf/proto"
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
	r.GET("/runtime", PageRuntime)

	return r
}

func getSocketTimeout() time.Duration {
	var t = 120
	if gopsu.IsExist(".sockettimeout") {
		b, err := ioutil.ReadFile(".sockettimeout")
		if err == nil {
			t = gopsu.String2Int(string(b), 10)
		}
	}
	if t < 120 {
		t = 120
	}
	return time.Second * time.Duration(t)
}

func getRoutes(h *gin.Engine) string {
	var sss string
	for _, v := range h.Routes() {
		if strings.ContainsAny(v.Path, "*") && !strings.HasSuffix(v.Path, "filepath") {
			return ""
		}
		if v.Path == "/" || v.Method == "HEAD" || strings.HasSuffix(v.Path, "*filepath") {
			continue
		}
		sss += fmt.Sprintf(`<a>%s: %s</a><br><br>`, v.Method, v.Path)
	}
	return sss
}

// ListenAndServe 启用监听
// port：端口号
// h： http.hander, like gin.New()
func ListenAndServe(port int, h *gin.Engine) error {
	sss := getRoutes(h)
	if sss != "" {
		h.GET("/showroutes", func(c *gin.Context) {
			c.Header("Content-Type", "text/html")
			c.Status(http.StatusOK)
			render.WriteString(c.Writer, sss, nil)
		})
	}
	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		ReadTimeout:  getSocketTimeout(),
		WriteTimeout: getSocketTimeout(),
		IdleTimeout:  getSocketTimeout(),
	}
	fmt.Fprintf(gin.DefaultWriter, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Success start HTTP server at :"+strconv.Itoa(port))
	return s.ListenAndServe()
}

// ListenAndServeTLS 启用TLS监听
// port：端口号
// h： http.hander, like gin.New()
// certfile： cert file path
// keyfile： key file path
// clientca: 客户端根证书用于验证客户端合法性
func ListenAndServeTLS(port int, h *gin.Engine, certfile, keyfile string, clientca ...string) error {
	if gopsu.IsExist(".forcehttp") {
		return ListenAndServe(port, h)
	}
	sss := getRoutes(h)
	if sss != "" {
		h.GET("/showroutes", func(c *gin.Context) {
			c.Header("Content-Type", "text/html")
			c.Status(http.StatusOK)
			render.WriteString(c.Writer, sss, nil)
		})
	}
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

	fmt.Fprintf(gin.DefaultWriter, "%s [90] [%s] %s\n", time.Now().Format(gopsu.ShortTimeFormat), "HTTP", "Success start HTTPS server at :"+strconv.Itoa(port))
	return s.ListenAndServeTLS(certfile, keyfile)
}

// CheckRequired 检查必填参数
func CheckRequired(params ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, v := range params {
			if c.Param(v) == "" {
				c.Set("status", 0)
				c.Set("detail", "Missing parameters: "+v)
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
		var ct = strings.Split(c.GetHeader("Content-Type"), ";")[0]
		var x = url.Values{}
		switch ct {
		case "multipart/form-data": // 文件上传
			x, _ = url.ParseQuery(c.Request.URL.RawQuery)
		case "", "application/json", "application/x-www-form-urlencoded": // 传参类，进行解析
			switch c.Request.Method {
			case "GET": // get请求忽略body内容
				x, _ = url.ParseQuery(c.Request.URL.RawQuery)
			default: // post，put，delete等请求只认body
				b, _ := ioutil.ReadAll(c.Request.Body)
				if len(b) > 0 {
					switch ct {
					case "", "application/x-www-form-urlencoded":
						x, _ = url.ParseQuery(string(b))
					default:
						c.Params = append(c.Params, gin.Param{
							Key:   "_raw",
							Value: string(b),
						})
						gjson.ParseBytes(b).ForEach(func(key, value gjson.Result) bool {
							x.Add(key.String(), value.String())
							return true
						})
					}
				}
			}
		}
		if len(x.Encode()) > 0 {
			c.Params = append(c.Params, gin.Param{
				Key:   "_raw",
				Value: x.Encode(),
			})
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
							Key:   "_cacheData",
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
						b, _ := proto.Marshal(ans)
						c.Params = append(c.Params, gin.Param{
							Key:   "_cacheData",
							Value: string(b),
						})
					}
				}
			}
		}
		c.Next()
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
			c.AbortWithStatusJSON(200, c.Keys)
			return
		}
		c.Next()
	}
}
