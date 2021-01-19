package main

import (
	"fmt"
	"time"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/db"
)

var (
	h   = -0.833
	uto = 180
)

// 启动文件 main.go
func main() {
	mysql := &db.SQLPool{
		User:         "root",
		Server:       "192.168.50.83",
		Passwd:       "lp1234xy",
		DataBase:     "v5db_eventlog",
		EnableCache:  false,
		MaxOpenConns: 200,
		CacheDir:     gopsu.DefaultCacheDir,
		Timeout:      120,
		DriverType:   db.DriverMYSQL,
	}
	mysql.New()
	println("start...")
	t1 := time.Now().UnixNano()
	query, err := mysql.QueryPB2New("select a.log_id,a.log_time,a.event_time,a.src_server,a.src_ip,a.user_name,a.asset_id,a.event_id,a.event_detail,a.event_status,a.event_point,a.loop_id,b.event_name from event_record as a left join event_info as b on a.event_id=b.event_id", 20)
	if err != nil {
		println(err.Error())
		return
	}
	t2 := time.Now().UnixNano()
	println(fmt.Sprintf("querypb2new: %.06f, %d", float32(t2-t1)/1000000000.0, query.Total))
	println(time.Now().UnixNano())
	// t1 = time.Now().UnixNano()
	// query, err = mysql.QueryPB2("select a.log_id,a.log_time,a.event_time,a.src_server,a.src_ip,a.user_name,a.asset_id,a.event_id,a.event_detail,a.event_status,a.event_point,a.loop_id,b.event_name from event_record as a left join event_info as b on a.event_id=b.event_id", 20)
	// if err != nil {
	// 	println(err.Error())
	// 	return
	// }
	// t2 = time.Now().UnixNano()
	// println(fmt.Sprintf("querypb2: %.06f, %d", float32(t2-t1)/1000000000.0, query.Total))
}
