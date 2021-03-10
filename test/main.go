package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/xyzj/gopsu/gin-middleware"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	r := ginmiddleware.NewGinEngine("log", "test.log", 1)
	r.Static("/ttt", "log")
	r.GET("/500", func(c *gin.Context) {
		panic(fmt.Errorf("format string, a ...interface{}"))
	})
	r.Run(":8080")
}
