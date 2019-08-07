package ginmiddleware

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

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
