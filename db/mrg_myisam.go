package db

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// only support mysql driver

// ShowTableInfo 查询表信息
// 返回最新子表的 名称，引擎名称，大小（MB），行数
func (p *SQLPool) ShowTableInfo(tableName string) (string, string, int64, int64, error) {
	var subTableName, engine string
	var tableSize, rowsCount int64
	if p.DriverType != DriveMySQL {
		return subTableName, engine, tableSize, rowsCount, errors.New("this function only support mysql driver")
	}
	if p.connPool == nil {
		return subTableName, engine, tableSize, rowsCount, errors.New("sql connection is not ready")
	}
	// 查询引擎
	strsql := "select engine,count(*) from information_schema.tables where table_schema=? and table_name=?"
	ans, err := p.QueryPB2(strsql, 1, p.DataBase, tableName)
	if err != nil {
		return subTableName, engine, tableSize, rowsCount, err
	}
	engine = ans.Rows[0].Cells[0]
	if strings.ToLower(engine) != "mrg_myisam" {
		return subTableName, engine, tableSize, rowsCount, errors.New("engine " + engine + " is not support")
	}
	// 获取最新子表
	strsql = "show create table " + tableName
	ans, err = p.QueryPB2(strsql, 1)
	if err != nil || len(ans.Rows) > 0 {
		return subTableName, engine, tableSize, rowsCount, err
	}
	s := ans.Rows[0].Cells[1] // gjson.Parse(ans).String()
	idx := strings.Index(s, "`"+tableName+"_")
	idx2 := strings.Index(s[idx+1:], "`")
	subTableName = s[idx+1 : idx+idx2+1]
	// 获取子表大小
	strsql = "select round(sum(DATA_LENGTH/1000000),2) as data,TABLE_ROWS,count(*) from information_schema.tables where table_schema=? and table_name=?"
	ans, err = p.QueryPB2(strsql, 1, p.DataBase, subTableName)
	if err != nil || len(ans.Rows) > 0 {
		return subTableName, engine, tableSize, rowsCount, err
	}
	tableSize, _ = strconv.ParseInt(ans.Rows[0].Cells[0], 10, 64)
	rowsCount, _ = strconv.ParseInt(ans.Rows[0].Cells[1], 10, 64)
	// tableSize = gopsu.String2Int64(ans.Rows[0].Cells[0], 10)
	// rowsCount = gopsu.String2Int64(ans.Rows[0].Cells[1], 10)
	// tableSize = int64(gjson.Parse(ans).Get("row.0").Float())
	// 获取子表行数
	// strsql = "select count(*) from " + subTableName
	// ans, err = p.QueryOne(strsql, 1)
	// if err != nil {
	// 	return subTableName, engine, tableSize, rowsCount, err
	// }
	// rowsCount = gjson.Parse(ans).Get("row.0").Int()
	return subTableName, engine, tableSize, rowsCount, nil
}

// MergeTable 进行分表操作
func (p *SQLPool) MergeTable(tableName string, maxSubTables int) error {
	if maxSubTables < 1 {
		return errors.New("maxsubTables should be more than 1")
	}
	if p.DriverType != DriveMySQL {
		return errors.New("this function only support mysql driver")
	}
	if p.connPool == nil {
		return errors.New("sql connection is not ready")
	}
	// 获取所有子表
	strsql := "select table_name from information_schema.tables where table_schema=? and table_name like '%" + tableName + "_%' order by table_name desc"
	ans, err := p.QueryPB2(strsql, 0, p.DataBase)
	if err != nil {
		return err
	}
	subTablelist := make([]string, maxSubTables)
	i := 0
	for _, row := range ans.Rows {
		subTablelist[i] = row.Cells[0]
		i++
		if i >= maxSubTables {
			break
		}
	}
	// gjson.Parse(ans).Get("rows").ForEach(func(key, value gjson.Result) bool {
	// 	subTablelist[i] = value.Get("cells.0").String()
	// 	i++
	// 	return i != maxSubTables
	// })
	if i == 0 {
		return errors.New("no sub tables found")
	}
	// 创建新子表
	subTableLatest := tableName + "_" + time.Now().Format("20060102150405")
	strsql = "create table " + subTableLatest + " like " + subTablelist[0]
	_, _, err = p.Exec(strsql)
	if err != nil {
		return errors.New("create new table " + subTableLatest + " error: " + err.Error())
	}
	// 修改总表
	strsql = "ALTER TABLE " + tableName + " ENGINE = MRG_MyISAM DEFAULT CHARSET=utf8 INSERT_METHOD=FIRST UNION=(" + subTableLatest
	for k, v := range subTablelist {
		if k == maxSubTables-1 || v == "" {
			break
		}
		strsql += "," + v
	}
	strsql += ")"
	_, _, err = p.Exec(strsql)
	if err != nil {
		return errors.New("alter table " + tableName + " error: " + err.Error())
	}
	return nil
}
