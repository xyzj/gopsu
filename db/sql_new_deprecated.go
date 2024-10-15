package db

import (
	"context"
	"errors"
	"strings"
)

// ExecTx 执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用事务
//
// Deprecated: use Exec()
func (d *Conn) ExecTx(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	return d.Exec(s, params...)
}

// ExecV2 事务执行语句（insert，delete，update），可回滚,返回（影响行数,insertId,error）,使用官方的语句参数分离写法
//
// Deprecated: use Exec()
func (d *Conn) ExecV2(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	return d.Exec(s, params...)
}

// ExecPrepareV2 批量执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用官方的语句参数分离写法，用于批量执行相同语句
//
// Deprecated: use ExecPrepare()
func (d *Conn) ExecPrepareV2(s string, paramNum int, params ...interface{}) (int64, []int64, error) {
	defer func() {
		if err := recover(); err != nil {
			d.cfg.Logger.Error("[DB] ExecPrepareV2 Err: " + err.(error).Error())
		}
	}()
	if paramNum == 0 {
		paramNum = strings.Count(s, "?")
	}

	l := len(params)
	if l%paramNum != 0 {
		return 0, nil, errors.New("not enough params")
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	defer cancel()
	// 开启事务
	sqldb, err := d.SQLDB(d.defaultDB)
	if err != nil {
		return 0, []int64{}, err
	}
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer d.rollbackCheck(tx)
	rowAffected := int64(0)
	insertID := make([]int64, len(params)/paramNum)
	idx := 0
	for i := 0; i < l; i += paramNum {
		ans, err := tx.ExecContext(ctx, s, params[i:i+paramNum]...)
		if err != nil {
			return 0, nil, err
		}
		rows, err := ans.RowsAffected()
		if err == nil {
			rowAffected += rows
		}
		inid, err := ans.LastInsertId()
		if err == nil {
			insertID[idx] = inid
		}
		idx++
	}
	err = tx.Commit()
	if err != nil {
		return 0, nil, err
	}
	return rowAffected, insertID, nil
}

// QueryJSON 执行查询语句，返回结果集的json字符串
//
// Deprecated: use Query()
func (d *Conn) QueryJSON(s string, rowsCount int, params ...interface{}) (string, error) {
	x, err := d.Query(s, rowsCount, params...)
	if err != nil {
		return "", err
	}
	return x.JSON()
}

// QueryOne 执行查询语句，返回首行结果的json字符串，`{row：[...]}`，该方法不缓存结果
//
// Deprecated: use Query() or QueryFirstPage()
func (d *Conn) QueryOne(s string, colNum int, params ...interface{}) (js string, err error) {
	pb, err := d.QueryFirstPageByDB(d.defaultDB, s, 1, params...)
	if err != nil {
		return "", err
	}
	if len(pb.Rows) == 0 {
		return "", nil
	}
	ss := pb.Rows[0].Cells
	if len(ss) == 0 {
		return "", nil //`{"row":[]}`, nil
	}
	return `{"row":["` + strings.Join(ss, "\",\"") + `"]}`, nil
}

// QueryOnePB2 执行查询语句，返回首行结果的QueryData结构，该方法不缓存结果
//
// Deprecated: use Query() or QueryFirstPage()
func (d *Conn) QueryOnePB2(s string, colNum int, params ...interface{}) (query *QueryData, err error) {
	qd, err := d.QueryFirstPageByDB(d.defaultDB, s, 1, params...)
	if err != nil {
		return qd, err
	}
	qd.Total = len(qd.Rows)
	return qd, nil
}

// QueryPB2 执行查询语句，返回QueryData结构
//
// Deprecated: use Query()
func (d *Conn) QueryPB2(s string, rowsCount int, params ...interface{}) (query *QueryData, err error) {
	return d.Query(s, rowsCount, params...)
}

// QueryPB2Chan 查询v2,采用线程+channel优化超大数据集分页的首页返回时间
//
// Deprecated: use Query() or QueryFirstPage() or QueryBig()
func (d *Conn) QueryPB2Chan(s string, rowsCount int, params ...interface{}) <-chan *QueryDataChan {
	ch := make(chan *QueryDataChan, 1)
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	go d.queryDataChan(ctx, cancel, d.dbs[d.defaultDB].sqldb, ch, s, rowsCount, params...)
	return ch
}

// QueryCachePB2 查询缓存结果，返回QueryData结构
//
// Deprecated: use QueryCache()
func (d *Conn) QueryCachePB2(cacheTag string, startRow, rowsCount int) *QueryData {
	return d.QueryCache(cacheTag, startRow, rowsCount)
}

// QueryCacheJSON 查询缓存结果，返回json字符串
//
// Deprecated: use QueryCache()
func (d *Conn) QueryCacheJSON(cacheTag string, startRow, rowsCount int) string {
	x, _ := d.QueryCache(cacheTag, startRow, rowsCount).JSON()
	return x
}
