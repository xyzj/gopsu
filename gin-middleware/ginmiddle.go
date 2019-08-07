package ginmiddleware

import (
	"net/url"

	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
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
