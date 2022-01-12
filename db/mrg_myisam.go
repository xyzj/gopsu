package db

import (
	fmt "fmt"
	"strings"
	"time"

	"github.com/xyzj/gopsu"

	"github.com/tidwall/gjson"
)

// only support mysql driver

// ShowTableInfo 查询表信息
// 返回最新子表的 名称，引擎名称，大小（MB），行数
func (p *SQLPool) ShowTableInfo(tableName string) (string, string, int64, int64, error) {
	var subTableName, engine string
	var tableSize, rowsCount int64
	if p.DriverType != DriverMYSQL {
		return subTableName, engine, tableSize, rowsCount, fmt.Errorf("this function only support mysql driver")
	}
	if p.connPool == nil {
		return subTableName, engine, tableSize, rowsCount, fmt.Errorf("sql connection is not ready")
	}
	// 查询引擎
	strsql := "select engine,count(*) from information_schema.tables where table_schema=? and table_name=?"
	ans, err := p.QueryOnePB2(strsql, 1, p.DataBase, tableName)
	if err != nil {
		return subTableName, engine, tableSize, rowsCount, err
	}
	engine = ans.Rows[0].Cells[0]
	if strings.ToLower(engine) != "mrg_myisam" {
		return subTableName, engine, tableSize, rowsCount, fmt.Errorf("engine %s is not support", engine)
	}
	// 获取最新子表
	strsql = "show create table " + tableName
	ans, err = p.QueryOnePB2(strsql, 2)
	if err != nil || len(ans.Rows) > 0 {
		return subTableName, engine, tableSize, rowsCount, err
	}
	s := ans.Rows[0].Cells[1] // gjson.Parse(ans).String()
	idx := strings.Index(s, "`"+tableName+"_")
	idx2 := strings.Index(s[idx+1:], "`")
	subTableName = s[idx+1 : idx+idx2+1]
	// 获取子表大小
	strsql = "select round(sum(DATA_LENGTH/1024/1024),2) as data,TABLE_ROWS,count(*) from information_schema.tables where table_schema=? and table_name=?"
	ans, err = p.QueryOnePB2(strsql, 1, p.DataBase, subTableName)
	if err != nil || len(ans.Rows) > 0 {
		return subTableName, engine, tableSize, rowsCount, err
	}
	tableSize = gopsu.String2Int64(ans.Rows[0].Cells[0], 10)
	rowsCount = gopsu.String2Int64(ans.Rows[0].Cells[1], 10)
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
		return fmt.Errorf("maxsubTables should be more than 1")
	}
	if p.DriverType != DriverMYSQL {
		return fmt.Errorf("this function only support mysql driver")
	}
	if p.connPool == nil {
		return fmt.Errorf("sql connection is not ready")
	}
	// 获取所有子表
	strsql := "select table_name from information_schema.tables where table_schema=? and table_name like '%" + tableName + "_%' order by table_name desc"
	ans, err := p.QueryJSON(strsql, 0, p.DataBase)
	if err != nil {
		return err
	}
	subTablelist := make([]string, maxSubTables)
	i := 0
	gjson.Parse(ans).Get("rows").ForEach(func(key, value gjson.Result) bool {
		subTablelist[i] = value.Get("cells.0").String()
		i++
		return i != maxSubTables
	})
	if i == 0 {
		return fmt.Errorf("no sub tables found")
	}
	// 创建新子表
	subTableLatest := fmt.Sprintf("%s_%d", tableName, time.Now().Unix())
	strsql = fmt.Sprintf("create table %s like %s", subTableLatest, subTablelist[0])
	_, _, err = p.Exec(strsql)
	if err != nil {
		return fmt.Errorf("create new table %s error: %s", subTableLatest, err.Error())
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
		return fmt.Errorf("alter table %s error: %s", tableName, err.Error())
	}
	return nil
}
