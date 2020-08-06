package main

import (
	"fmt"
	"time"

	"github.com/xyzj/gopsu/db"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	conn := &db.SQLPool{
		Server: "192.168.50.83",
		// 用户名
		User: "root",
		// 密码
		Passwd: "lp1234xy",
		// 数据库名称
		DataBase: "projectall",
		// 数据驱动
		DriverType:  db.DriverMYSQL,
		EnableCache: true,
	}
	conn.New()
	data, err := conn.QueryMultirowPage("select * from project_record_view", 2, 0)
	if err != nil {
		println(err.Error())
		return
	}
	println(fmt.Sprintf("%+v", data))
	cacheTag := data.CacheTag
	time.Sleep(time.Second)
	dataPage := conn.QueryCacheMultirowPage(cacheTag, 2, 2, 0)
	println(fmt.Sprintf("%+v", dataPage))
}
