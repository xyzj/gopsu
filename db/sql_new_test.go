package db

import (
	"os"
	"strings"
	"testing"

	"github.com/xyzj/gopsu/logger"
)

func TestInit(t *testing.T) {
	b, _ := os.ReadFile("dbinit.sql")
	opt := &Opt{
		Server:      "192.168.50.83:13306",
		User:        "root",
		Passwd:      "lp1234xy",
		DBNames:     []string{"v5db_test_mrg", "dba2"},
		InitScripts: []string{string(b)},
		DriverType:  DriveMySQL,
		Logger:      logger.NewConsoleLogger(),
	}
	a, err := New(opt)
	if err != nil {
		t.Fatal(err)
		return
	}
	println(a.IsReady())
}

func TestUpg(t *testing.T) {
	b, _ := os.ReadFile("dbupg.sql")
	opt := &Opt{
		Server:     "192.168.50.83:13306",
		User:       "root",
		Passwd:     "lp1234xy",
		DBNames:    []string{"v5db_test_mrg", "dba2"},
		DriverType: DriveMySQL,
		Logger:     logger.NewConsoleLogger(),
	}
	a, err := New(opt)
	if err != nil {
		t.Fatal(err)
		return
	}
	for _, s := range strings.Split(string(b), ";|") {
		if strings.TrimSpace(s) == "" {
			continue
		}
		a.Exec(s)
	}
}
func TestMrg(t *testing.T) {
	opt := &Opt{
		Server:     "192.168.50.83:13306",
		User:       "root",
		Passwd:     "lp1234xy",
		DBNames:    []string{"v5db_test_mrg", "dba2"},
		DriverType: DriveMySQL,
		Logger:     logger.NewConsoleLogger(),
	}
	a, err := New(opt)
	if err != nil {
		t.Fatal(err)
		return
	}
	err = a.MergeTable("v5db_test_mrg", "event_record", 10, 10, 10)
	if err != nil {
		t.Fatal(err)
		return
	}
}

func TestQuery(t *testing.T) {
	opt := &Opt{
		Server:     "192.168.50.83:13306",
		User:       "root",
		Passwd:     "lp1234xy",
		DBNames:    []string{"v5db_assetdatacenter"},
		DriverType: DriveMySQL,
		Logger:     logger.NewConsoleLogger(),
	}
	a, err := New(opt)
	if err != nil {
		t.Fatal(err)
		return
	}
	ans, err := a.Query("select dt_create,dt_update from asset_info where aid=?", 0, "01011100059377714613")
	if err != nil {
		t.Fatal(err)
		return
	}
	println(ans.Rows[0].VCells[0].TryTimestamp(""), ans.Rows[0].VCells[1].TryTimestamp(""))
}
