package db

import (
	"fmt"
	"strings"
	"time"
)

func (d *Conn) UnionView(dbname, tableName string, maxSubTables, maxTableSize, maxTableRows int) error {
	// 判断是否符合分表要求
	if d.cfg.DriverType != DriveMySQL {
		return fmt.Errorf("this function only support mysql driver")
	}
	dbidx := 1
	for k, v := range d.dbs {
		if v.name == dbname {
			dbidx = k
		}
	}
	if !d.IsReady() {
		return fmt.Errorf("sql connection is not ready")
	}
	// 检查主表大小或数据量
	strsql := `select sum(DATA_LENGTH/1000000),sum(table_rows) from information_schema.tables where table_schema=? and table_name=?`
	ans, err := d.QueryByDB(dbidx, strsql, 0, dbname, tableName)
	if err != nil {
		return err
	}
	if ans.Rows[0].VCells[0].TryInt() < maxTableSize && ans.Rows[0].VCells[1].TryInt() < maxTableRows {
		return nil
	}
	//将主表重命名为日期后缀子表
	newTableName := tableName + "_" + time.Now().Format("200601021504")
	strsql = fmt.Sprintf("rename table %s to %s", tableName, newTableName)
	_, _, err = d.ExecByDB(dbidx, strsql)
	if err != nil {
		return err
	}

	// 找到所有以指定命名开头的所有表
	strsql = "select table_name from information_schema.tables where table_schema=? and table_name like '%" + tableName + "_%' order by table_name desc"
	ans, err = d.QueryByDB(dbidx, strsql, 0, d.defaultDB)
	if err != nil {
		return err
	}
	subTablelist := make([]string, 0, maxSubTables)
	i := 0
	for _, row := range ans.Rows {
		subTablelist = append(subTablelist, row.Cells[0])
		i++
		if i >= maxSubTables {
			break
		}
	}

	// 创建新的空主表
	strsql = fmt.Sprintf("create table %s like %s", tableName, newTableName)
	_, _, err = d.ExecByDB(dbidx, strsql)
	if err != nil {
		return err
	}
	// 修改视图，加入新子表或以及判断删除旧子表
	strsql = fmt.Sprintf(`CREATE OR REPLACE ALGORITHM = MERGE VIEW %s_view AS
	select * from %s `, tableName, tableName)
	for _, s := range subTablelist {
		strsql += " union select * from " + s
	}
	_, _, err = d.ExecByDB(dbidx, strsql)
	if err != nil {
		return err
	}
	return nil
}

// MergeTable 进行分表操作
func (d *Conn) MergeTable(dbname, tableName string, maxSubTables, maxTableSize, maxTableRows int) error {
	if d.cfg.DriverType != DriveMySQL {
		return fmt.Errorf("this function only support mysql driver")
	}
	dbidx := 1
	for k, v := range d.dbs {
		if v.name == dbname {
			dbidx = k
			break
		}
	}
	// 查询引擎
	strsql := "select engine from information_schema.tables where table_schema=? and table_name=?"
	ans, err := d.QueryByDB(dbidx, strsql, 1, dbname, tableName)
	if err != nil {
		return err
	}
	engine := ans.Rows[0].Cells[0]
	if strings.ToLower(engine) != "mrg_myisam" {
		return fmt.Errorf("engine " + engine + " is not support")
	}
	// 找到所有以指定命名开头的所有表
	strsql = "select table_name from information_schema.tables where table_schema=? and table_name like '%" + tableName + "_%' order by table_name desc limit ?"
	ans, err = d.QueryByDB(dbidx, strsql, 0, dbname, maxSubTables)
	if err != nil {
		return err
	}
	subTablelist := make([]string, 0)
	for _, row := range ans.Rows {
		subTablelist = append(subTablelist, row.Cells[0])
	}
	if len(subTablelist) == 0 {
		return fmt.Errorf("no sub tables found")
	}
	// 检查子表大小
	strsql = `select sum(DATA_LENGTH/1000000),sum(table_rows) from information_schema.tables where table_schema=? and table_name=?`
	ans, err = d.QueryByDB(dbidx, strsql, 1, dbname, subTablelist[0])
	if err != nil {
		return err
	}
	if ans.Rows[0].VCells[0].TryInt() < maxTableSize && ans.Rows[0].VCells[1].TryInt() < maxTableRows {
		return nil
	}
	// 创建新子表
	subTableLatest := tableName + "_" + time.Now().Format("060102150405")
	strsql = "create table " + subTableLatest + " like " + subTablelist[0]
	_, _, err = d.ExecByDB(dbidx, strsql)
	if err != nil {
		return fmt.Errorf("create new table " + subTableLatest + " error: " + err.Error())
	}
	newsub := make([]string, 0)
	newsub = append(newsub, subTableLatest)
	newsub = append(newsub, subTablelist...)
	// 修改总表
	strsql = "ALTER TABLE " + tableName + " ENGINE = MRG_MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci INSERT_METHOD=FIRST UNION=(" + strings.Join(newsub, ",") + ");"
	_, _, err = d.ExecByDB(dbidx, strsql)
	if err != nil {
		return fmt.Errorf("alter table " + tableName + " error: " + err.Error())
	}
	return nil
}
