package main

import (
	"fmt"
	"time"

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
		DataBase:    "mydb10001",
		Logger:      &gopsu.StdLogger{},
		Timeout:     10,
	}
	err := sql.New()
	if err != nil {
		println("conn", err.Error())
		return
	}
	strsql := "select * from user_rwx"
	strsql = "truncate table user_rwx"
	sql.Exec(strsql)
	strsql = "insert into user_rwx set user_name=?,r=?,w=?,x=?,d=?"
	params := make([]interface{}, 5000)
	for i := 0; i < 5000; i += 5 {
		params[i] = fmt.Sprintf("user%d", i)
		params[i+1] = 0
		params[i+2] = 0
		params[i+3] = 0
		params[i+4] = 0
	}
	t1 := time.Now()
	err = sql.ExecPrepare(strsql, 0, params...)
	if err != nil {
		println(err.Error())
	}
	t2 := time.Now()
	strsql = "select count(*) from user_rwx"
	ans, _ := sql.QueryOne(strsql, 1)
	println(ans)
	println(t2.Sub(t1).Nanoseconds() / 1000000)
	// s, _ = sql.QueryJSON("select * from user_rwx", 0)
	// println(s)
}
