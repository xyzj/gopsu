package main

import (
	"time"

	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
	db "github.com/xyzj/gopsu/db"
)

func main() {
	sql := db.SQLPool{
		Server:      "180.153.108.83:3306",
		User:        "root",
		Passwd:      "lp1234xy",
		CacheDir:    ".",
		EnableCache: true,
		DriverType:  db.DriverMYSQL,
		DataBase:    "mydb10001_data",
		Logger:      &gopsu.StdLogger{},
		Timeout:     100,
	}
	err := sql.New()
	if err != nil {
		println("conn", err.Error())
		return
	}
	t1 := time.Now().Unix()
	strsql := "select * from data_rtu_record"
	s, err := sql.QueryJSON(strsql, 0)
	if err != nil {
		println("query", err.Error())
		return
	}
	t2 := time.Now().Unix()

	println(t2-t1, gjson.Parse(s).Get("total").Int())
	// s, _ = sql.QueryJSON("select * from user_rwx", 0)
	// println(s)
}
