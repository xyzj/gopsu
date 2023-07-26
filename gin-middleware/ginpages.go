package ginmiddleware

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xyzj/gopsu"
)

//go:embed favicon.webp
var favicon []byte

//go:embed pages/404-big.html
var page404big []byte

//go:embed pages/404.html
var page404 []byte

//go:embed pages/500.html
var page500 []byte

//go:embed pages/403.html
var page403 []byte

//go:embed pages/403-city.html
var page403City []byte

//go:embed pages/helloworld.html
var pageHelloworld []byte

var (
	templateEmpty = []byte(`<p><span style="color:hsl(0,0%,100%);"><strong>If you don't know what you're doing, just walk away...</strong></span></p>`)
)

// PageEmpty PPageEmptyage403
func PageEmpty(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Write(templateEmpty)
}

// PageAbort PPageEmptyage403
func PageAbort(c *gin.Context) {
	c.AbortWithStatus(http.StatusGone)
}

// Page403 Page403
func Page403(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.Header("Content-Type", "text/html")
		c.Writer.WriteHeader(http.StatusForbidden)
		c.Writer.Write(page403)
		return
	}
	c.String(http.StatusForbidden, "403 Forbidden")
}

// Page404 Page404
func Page404(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.Header("Content-Type", "text/html")
		c.Writer.WriteHeader(http.StatusNotFound)
		c.Writer.Write(page404)
		return
	}
	c.String(http.StatusNotFound, "404 nothing here")
}

// Page404Big Page404
func Page404Big(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.Header("Content-Type", "text/html")
		c.Writer.WriteHeader(http.StatusNotFound)
		c.Writer.Write(page404big)
		return
	}
	c.String(http.StatusNotFound, "404 nothing here")
}

// Page405 Page405
func Page405(c *gin.Context) {
	c.String(http.StatusMethodNotAllowed, "405 "+c.Request.Method+" is not allowed")
}

// PageDev PageDev
func PageDev(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.Writer.WriteHeader(http.StatusServiceUnavailable)
	c.Writer.Write(page500)
}

// PageDefault 健康检查
func PageDefault(c *gin.Context) {
	switch c.Request.Method {
	case "GET":
		if c.Request.RequestURI == "/" {
			c.Header("Content-Type", "text/html")
			c.Writer.WriteHeader(http.StatusOK)
			c.Writer.Write(pageHelloworld)
		} else {
			c.String(http.StatusOK, "ok")
		}
	case "POST":
		c.String(http.StatusOK, "ok")
	}
}

// Clearlog 日志清理
func Clearlog(c *gin.Context) {
	if c.Param("pwd") != "xyissogood" {
		c.String(200, "Wrong!!!")
		return
	}
	var days int64
	if days = gopsu.String2Int64(c.Param("days"), 0); days == 0 {
		days = 7
	}
	// 遍历文件夹
	dir := c.Param("dir")
	if dir == "" {
		dir = gopsu.DefaultLogDir
	}
	lstfno, ex := os.ReadDir(dir)
	if ex != nil {
		os.WriteFile("ginlogerr.log", gopsu.Bytes(fmt.Sprintf("clear log files error: %s", ex.Error())), 0664)
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
		if !strings.Contains(fno.Name(), c.Param("name")) {
			continue
		}
		// 比对文件生存期
		if t.Unix()-fno.ModTime().Unix() >= days*24*60*60-10 {
			os.Remove(filepath.Join(c.Param("dir"), fno.Name()))
			c.Set(fno.Name(), "deleted")
		}
	}
	c.JSON(200, c.Keys)
}
