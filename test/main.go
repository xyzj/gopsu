package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xyzj/gopsu"
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
func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	// binary.BigEndian.PutUint64(bytes, bits)
	binary.LittleEndian.PutUint64(bytes, bits)

	return bytes
}

// 启动文件 main.go
func main() {
	a := 12345678.90
	b := gopsu.Float64ToByte(a)
	println(gopsu.Bytes2String(b, "-"))
	c := gopsu.Bytes2Float64(b, false)
	println(fmt.Sprintf("%f", c))
	bits := binary.LittleEndian.Uint64(b)
	println(bits, gopsu.Bytes2Uint64(b, false))
}
