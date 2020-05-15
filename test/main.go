package main

import (
	"github.com/xyzj/gopsu/db"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	mysqlClient := &db.SQLPool{
		User:        "root",
		Server:      "192.168.50.83",
		Passwd:      "lp1234xy",
		DataBase:    "sample_eventlog",
		EnableCache: false,
		CacheDir:    "",
		CacheHead:   "test",
		Timeout:     120,
		Logger:      nil,
		DriverType:  db.DriverMYSQL,
	}
	mysqlClient.New()
	tableName := "event_record"
	strsql := "select engine from information_schema.tables where table_schema=? and table_name=?"
	ans, err := mysqlClient.QueryOne(strsql, 1, "sample_eventlog", tableName)
	if err != nil {
		println("err: ", err.Error())
		return
	}
	println(ans)
	strsql = "show create table " + tableName
	ans, err = mysqlClient.QueryOne(strsql, 2)
	// _, _, _, _, err := mysqlClient.ShowTableInfo("event_record")
	if err != nil {
		println("err: ", err.Error())
		return
	}
	println(ans)
}
