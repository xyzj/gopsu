package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/xyzj/gopsu/gin-middleware"
	wmv2 "github.com/xyzj/wlstmicro/v2"
)

var (
	fw = wmv2.NewFrameWorkV2(`{"ver":"v1.0.0"}`)
)

func GetAvailablePort() (int, error) {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", "0.0.0.0"))
	if err != nil {
		return 0, err
	}
	var listener *net.TCPListener
	var found = false
	for i := 0; i < 100; i++ {
		listener, err = net.ListenTCP("tcp", address)
		if err != nil {
			continue
		}
		found = true
	}
	defer listener.Close()
	if !found {
		return 0, fmt.Errorf("could not find a useful port")
	}
	return listener.Addr().(*net.TCPAddr).Port, nil
}
func routesEngine() *gin.Engine {
	// fw.LoadConfigure()
	r := fw.NewHTTPEngine(ginmiddleware.ReadParams())
	r.GET("/test", test)
	return r
}
func test(c *gin.Context) {
	println("RequestURI", c.Request.RequestURI)
	println("uri requesturi", c.Request.URL.RequestURI())
	println("rawquery", c.Request.URL.RawQuery, c.Request.URL.RawFragment, c.Request.URL.RawPath)
	x, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		println("err", err.Error())
	}
	println(len(x))
	for k := range x {
		println(k, x.Get(k))
	}
	c.String(200, "format string")
}
func String2Int8(s string, t int) byte {
	x, _ := strconv.ParseUint(s, t, 8)
	return byte(x)
}
func String2Int80(s string, t int) byte {
	x, _ := strconv.ParseInt(s, t, 0)
	return byte(x)
}

// 启动文件 main.go
func main() {
	a := fmt.Sprintf("%b0%06b", 1, 0)
	a = "81a"
	a1 := String2Int8(a, 16)
	a2 := String2Int80(a, 16)
	println(a, fmt.Sprintf("%x, %x", a1, a2))

}
