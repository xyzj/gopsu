package main

import (
	"github.com/xyzj/gopsu"
	db "github.com/xyzj/gopsu/sql"
)

func main() {
	sql := db.SQLPool{
		Server:      "180.153.108.83:3306",
		User:        "root",
		Passwd:      "lp1234xy",
		CacheDir:    ".",
		EnableCache: true,
		DriverType:  db.DriverMYSQL,
		DataBase:    "mydb10001",
		Logger:      &gopsu.StdLogger{},
	}
	err := sql.New()
	if err != nil {
		println(err.Error())
		return
	}
	strsql := "select * from user_list"
	s, err := sql.QueryJSON(strsql, 0)
	if err != nil {
		println(err.Error())
		return
	}
	println(s)
	s, _ = sql.QueryJSON("select * from user_rwx", 0)
	println(s)
}
