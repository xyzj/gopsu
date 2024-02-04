package db

import (
	"testing"

	"github.com/goccy/go-json"
)

func TestSQL(t *testing.T) {
	s := SQLPool{
		Server:   "192.168.50.83:3306",
		User:     "root",
		Passwd:   "lp1234xy",
		DataBase: "v5db_assetdatacenter",
	}
	s.New("false")
	t.Run("query v2", func(t *testing.T) {
		ans, err := s.Query("select aid,name,dt_create from asset_info", 1000)
		if err != nil {
			println(err.Error())
			t.Fail()
			return
		}
		println(ans.Total)
	})
	t.Run("query one", func(t *testing.T) {
		ans, err := s.Query("select aid,name,dt_create from asset_info", 0)
		if err != nil {
			println(err.Error())
			t.Fail()
			return
		}
		println(len(ans.JSON()))
	})
}
func BenchmarkSQL(t *testing.B) {
	s := SQLPool{
		Server:   "192.168.50.83:3306",
		User:     "root",
		Passwd:   "lp1234xy",
		DataBase: "v5db_assetdatacenter",
	}
	s.New("false")
	ans, err := s.Query("select aid,name,dt_update from asset_info", 0)
	if err != nil {
		t.Fail()
		return
	}
	t.ResetTimer()
	t.Run("no escap", func(t *testing.B) {
		_, err := json.MarshalNoEscape(ans)
		if err != nil {
			t.Fail()
			return
		}
	})
	t.Run("with opt", func(t *testing.B) {
		_, err := json.MarshalWithOption(ans, json.UnorderedMap())
		if err != nil {
			t.Fail()
			return
		}
	})
}
