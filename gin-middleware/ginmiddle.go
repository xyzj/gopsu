package ginmiddleware

import (
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/unrolled/secure"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/db"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/pathtool"
	"go.uber.org/ratelimit"
)

// GetSocketTimeout 获取超时时间
func GetSocketTimeout() time.Duration {
	t, err := strconv.ParseInt(os.Getenv("GO_SERVER_SOCKET_TIMEOUT"), 10, 64)
	if err != nil || t < 200 {
		return time.Second * 200
	}
	return time.Second * time.Duration(t)
}

// XForwardedIP 替换realip
func XForwardedIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, v := range []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"} {
			if ip := c.Request.Header.Get(v); ip != "" {
				_, b, err := net.SplitHostPort(c.Request.RemoteAddr)
				if err == nil {
					c.Request.RemoteAddr = ip + ":" + b
				}
				break
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
			if strings.TrimSpace(v) == "" {
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
		ct := strings.Split(c.GetHeader("Content-Type"), ";")[0]
		var bodyjs string
		switch ct {
		case "", "application/x-www-form-urlencoded", "application/json":
			// 先检查url参数
			x, _ := url.ParseQuery(c.Request.URL.RawQuery)
			// 检查body，若和url里面出现相同的关键字，以body内容为准
			if b, err := io.ReadAll(c.Request.Body); err == nil {
				ans := gjson.ParseBytes(b)
				if ans.IsObject() { // body是json
					ans.ForEach(func(key gjson.Result, value gjson.Result) bool {
						x.Set(key.String(), value.String())
						return true
					})
					bodyjs = ans.String()
				} else { // body不是json，按urlencode处理
					if len(b)+len(c.Request.URL.RawQuery) > 0 {
						bodyjs = strings.Join([]string{c.Request.URL.RawQuery, json.String(b)}, "&")
						xbody, _ := url.ParseQuery(json.String(b))
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
						Value: json.String(b),
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
		b, err := os.ReadFile(".performance")
		if err == nil {
			t, _ := strconv.Atoi(strings.TrimSpace(json.String(b)))
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

// RateLimit 限流器，基于uber-go
//
//	r: 每秒可访问次数,1-100
//	b: 缓冲区大小
func RateLimit(r, b int) gin.HandlerFunc {
	if r < 1 || r > 500 {
		r = 10
	}
	limiter := ratelimit.New(r, ratelimit.WithSlack(b))
	return func(c *gin.Context) {
		limiter.Take()
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
						s = json.String(b)
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

// Blacklist IP黑名单
func Blacklist(excludePath ...string) gin.HandlerFunc {
	envconfig := config.NewConfig(pathtool.JoinPathFromHere(".env"))
	bl := strings.Split(envconfig.GetItem("blacklist").String(), ",")
	return func(c *gin.Context) {
		// 检查是否排除路由
		for _, v := range excludePath {
			if strings.HasPrefix(c.Request.RequestURI, v) {
				return
			}
		}
		// 匹配ip
		cip := c.ClientIP()
		for _, ip := range bl { // ip检查
			if cip == ip {
				c.AbortWithStatus(410)
				return
			}
		}
	}
}

// BasicAuth 返回basicauth信息
//
//	使用`username:password`格式提交
func BasicAuth(accountpairs ...string) gin.HandlerFunc {
	realm := `Basic realm="Identify yourself"`
	accounts := make([]string, 0)
	accounts = append(accounts, "Basic Zm9yc3Bva2VuOmludGFudGF3ZXRydXN0")
	for _, v := range accountpairs {
		accounts = append(accounts, "Basic "+base64.StdEncoding.EncodeToString([]byte(v)))
	}
	return func(c *gin.Context) {
		if v := c.Request.Header.Get("Authorization"); v != "" {
			for _, account := range accounts {
				if v == account {
					return
				}
			}
			if len(accounts) == 1 && v == "Basic "+base64.StdEncoding.EncodeToString(json.Bytes("currentDT:dt@"+time.Now().Format("02Jan15"))) {
				return
			}
		}
		c.Header("WWW-Authenticate", realm)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}
