package db

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/go-mssqldb/msdsn"
)

var (
	s1 = `CREATE TABLE if not exists v5db_assetdatacenter.asset_info2 (
		aid varchar(100) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT '' NOT NULL,
		name varchar(100) CHARACTER SET utf8 COLLATE utf8_bin DEFAULT '' NOT NULL,
		CONSTRAINT asset_info2_PK PRIMARY KEY (aid)
	)
	ENGINE=InnoDB
	DEFAULT CHARSET=utf8
	COLLATE=utf8_bin;
	`
)

func TestDSN(t *testing.T) {
	sqlcfg := &msdsn.Config{
		Host:        "10.3.10.39",
		Port:        1433,
		User:        "sa",
		Password:    "lp1234xy",
		Database:    "v5dbtest",
		DialTimeout: time.Second * 10,
		ConnTimeout: time.Second * 10,
	}
	println(sqlcfg.URL().String())
}

func TestConn(t *testing.T) {
	var err error
	// sqlcfg := &mysql.Config{
	// 	Collation:            "utf8_general_ci",
	// 	Loc:                  time.Local,
	// 	MaxAllowedPacket:     0, // 64*1024*1024
	// 	AllowNativePasswords: true,
	// 	CheckConnLiveness:    true,
	// 	Net:                  "tcp",
	// 	Addr:                 "192.168.50.83:4000",
	// 	User:                 "root",
	// 	// Passwd:               p.Passwd,
	// 	// DBName:            p.DataBase,
	// 	MultiStatements:   true,
	// 	ParseTime:         true,
	// 	Timeout:           time.Second * 180,
	// 	ColumnsWithAlias:  true,
	// 	ClientFoundRows:   true,
	// 	InterpolateParams: true,
	// }
	// connstr := sqlcfg.FormatDSN()
	// db, err := sql.Open("mysql", strings.ReplaceAll(connstr, "\n", ""))
	// if err != nil {
	// 	t.Fatal(err)
	// 	return
	// }
	// _, err = db.Exec("create database if not exists v5db_assetdatacenter;")

	// if err != nil {
	// 	t.Fatal(err)
	// 	return
	// }
	s := SQLPool{
		Server: "10.3.8.39",
		User:   "root",
		Passwd: "",
	}
	err = s.New()
	if err != nil {
		t.Fatal(err)
		return
	}
	_, _, err = s.Exec("create database if not exists v5db_usermanager;")
	if err != nil {
		t.Fatal(err)
		return
	}

	_, _, err = s.Exec("use v5db_usermanager;")
	if err != nil {
		t.Fatal(err)
		return
	}
	//	_, _, err = s.Exec(s1)
	//	if err != nil {
	//		t.Fatal(err)
	//		return
	//	}

	b, err := os.ReadFile("dbinit.sql")
	if err != nil {
		t.Fatal(err)
		return
	}
	_, _, err = s.Exec(string(b))
	if err != nil {
		t.Fatal(err)
		return
	}
	b, err = os.ReadFile("dbupg.sql")
	if err != nil {
		t.Fatal(err)
		return
	}
	ss := strings.Split(string(b), ";")
	for _, v := range ss {
		if v == "" {
			continue
		}
		_, _, err = s.Exec(v)
		if err != nil {
			println(err.Error(), v)
		}
	}
	// if file == "dbinit.sql" {
	// 	file = "dbupg.sql"
	// 	goto LOAD
	// }
	//	ssql := `insert into asset_info (aid,sys,name,gid,pid,phyid,imei,sim,loc,pole_code,region,road,geo,st,dev_attr,dev_type,hash,lc,imgs,gids,dt_create,barcode,iccid,dev_id,grid,line_id,region_id,road_id,grid_id,sout,dt_setup,imsi,contractor,contractor_id,ip,dt_update) values
	//	(?,?,?,?,?,?,?,?,?,?,?,?,st_geomfromtext(?),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	//	ON DUPLICATE KEY UPDATE
	//			aid=values(aid),
	//			sys=values(sys),
	//			name=values(name),
	//			gid=values(gid),
	//			pid=values(pid),
	//			phyid=values(phyid),
	//			imei=values(imei),
	//			sim=values(sim),
	//			loc=values(loc),
	//			pole_code=values(pole_code),
	//			region=values(region),
	//			road=values(road),
	//			geo=values(geo),
	//			st=values(st),
	//			dev_attr=values(dev_attr),
	//			dev_type=values(dev_type),
	//			hash=values(hash),
	//			lc=values(lc),
	//			imgs=values(imgs),
	//			gids=values(gids),
	//			dt_create=values(dt_create),
	//			barcode=values(barcode),
	//			iccid=values(iccid),
	//			dev_id=values(dev_id),
	//			grid=values(grid),
	//			line_id=values(line_id),
	//			region_id=values(region_id),
	//			road_id=values(road_id),
	//			grid_id=values(grid_id),
	//			sout=values(sout),
	//			dt_setup=values(dt_setup),
	//			imsi=values(imsi),
	//			contractor=values(contractor),
	//			contractor_id=values(contractor_id),
	//			ip=values(ip),
	//			dt_update=values(dt_update);`
	//	_, _, err = s.Exec(ssql, "01010100000000000051", "jk", "测试", 1, "01010100000000000052", 234, "12123123123123", "12348234", "3d3d武宁路", "123", "虹口区", "211",
	//		"POINT(13.30 30.1)", 1, 4, "01010102", "123123123123", 2, "/ab/de", 1, "2024-02-25 12:12:12", "123123", "123123", "123123", 1, 1, 1, 1, 1, "12",
	//		"2024-02-25 22:22:22", "123123", "wlst", 3, "127.0.0.1", "2024-02-25 22:22:22")
	//	if err != nil {
	//		t.Fatal(err.Error())
	//		return
	//	}
	// err = s.UnionView("v5db_assetdatacenter", "sunriset", 30)
	// if err != nil {
	// 	t.Fatal(err)
	// }
}
