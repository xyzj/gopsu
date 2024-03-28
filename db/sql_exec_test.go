package db

import "testing"

func TestExec(t *testing.T) {
	s := SQLPool{
		Server:   "192.168.50.83:3306",
		User:     "root",
		Passwd:   "lp1234xy",
		DataBase: "mydb1024",
	}
	s.New()
	t.Run("exec rollback", func(t *testing.T) {
		strsql := "insert into 1ab (t1,t2) values (?,?),(?,?);"
		rows, _, err := s.ExecV2(strsql, 67, "sdfsf", 32, "edf4d")
		if err != nil {
			t.Fatal(err.Error())
			return
		}
		println(rows)
	})
}
