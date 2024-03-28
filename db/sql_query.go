package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/crypto"
)

// QueryJSON 执行查询语句，返回结果集的json字符串
//
// s: sql语句
// rowsCount: 返回数据行数，0-返回全部
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryJSON(s string, rowsCount int, params ...interface{}) (string, error) {
	x, err := p.Query(s, rowsCount, params...)
	if err != nil {
		return "", err
	}
	return x.JSON(), nil
}

// QueryLimit 执行查询语句，限制返回行数
//
// s: sql语句
//
// startRow: 起始行号，0开始
//
// rowsCount: 返回数据行数，0-返回全部
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryLimit(s string, startRow, rowsCount int, params ...interface{}) (*QueryData, error) {
	if startRow+rowsCount == 0 {
		return p.Query(s, rowsCount, params...)
	}
	switch p.DriverType {
	case DriveSQLServer:
		s += fmt.Sprintf(" between %d and %d", startRow, startRow+rowsCount)
	case DriveMySQL:
		s += fmt.Sprintf(" limit %d,%d", startRow, rowsCount)
	}
	query, err := p.Query(s, 0, params...)
	if err != nil {
		return nil, err
	}
	query.CacheTag = emptyCacheTag
	return query, nil
}

// QueryBig 可尝试用于大数据集的首页查询，一定程度加快速度，benchmark测试没有where有索引的情况下比传统查询快10倍？？？
//
// s: sql语句
//
// startRow: 起始行号，0开始
//
// rowsCount: 返回数据行数，0-返回全部
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryBig(s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	if rowsCount == 0 {
		return p.Query(s, rowsCount, params...)
	}
	ss := "select count(*) " + s[strings.Index(s, "from"):]
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()
	var total int
	err := p.connPool.QueryRowContext(ctx, ss, params...).Scan(&total)
	switch {
	case err == sql.ErrNoRows:
		return newResult(), nil
	case err != nil:
		return p.Query(s, rowsCount, params...)
	default:
		qd, err := p.QueryFirstPage(s, rowsCount, params...)
		qd.Total = total
		return qd, err
	}
}

// QueryFirstPage 返回第一页，不返回总数，用于大数据集的首页查询，采用不定数量翻页
//
//	s: sql语句
//	rowsCount: 返回数据行数，0-返回全部
//	params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryFirstPage(s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	if rowsCount == 0 {
		return p.Query(s, rowsCount, params...)
	}
	var ch = make(chan *QueryDataChan, 1)
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	// p.chanQuery <- &QueryDataChanWorker{
	// 	QDC:       ch,
	// 	Done:      cancel,
	// 	Strsql:    s,
	// 	RowsCount: rowsCount,
	// 	Params:    params,
	// }
	go p.queryDataChan(ctx, cancel, ch, s, rowsCount, params...)
	select {
	case q := <-ch:
		return q.Data, q.Err
	case <-ctx.Done():
		return newResult(), fmt.Errorf("query data timeout")
	}
}

// Query 新的查询方法，只填充VCells
func (p *SQLPool) Query(s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	var ch = make(chan *QueryDataChan, 1)
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	var qd = newResult()
	var err error
	// p.chanQuery <- &QueryDataChanWorker{
	// 	QDC:       ch,
	// 	Done:      cancel,
	// 	Strsql:    s,
	// 	RowsCount: rowsCount,
	// 	Params:    params,
	// }
	go p.queryDataChan(ctx, cancel, ch, s, rowsCount, params...)
	var q *QueryDataChan
ANS:
	for {
		select {
		case q = <-ch:
			qd = q.Data
			err = q.Err
		case <-ctx.Done():
			qd.Total = *q.Total
			break ANS
		}
	}
	return qd, err
}

func (p *SQLPool) queryDataChan(ctx context.Context, done context.CancelFunc, ch chan *QueryDataChan, s string, rowsCount int, params ...interface{}) int {
	defer func() {
		if err := recover(); err != nil {
			ch <- &QueryDataChan{
				Data: newResult(),
				Err:  err.(error),
			}
		}
		done()
	}()

	if rowsCount < 0 {
		rowsCount = 0
	}
	var rowIdx = 0
	// 查询数据集
	rows, err := p.connPool.QueryContext(ctx, s, params...)
	if err != nil {
		ch <- &QueryDataChan{
			Data:  newResult(),
			Err:   err,
			Total: &rowIdx,
		}
		return 0
	}
	defer rows.Close()
	// 处理数据集
	columns, err := rows.Columns()
	if err != nil {
		ch <- &QueryDataChan{
			Data:  newResult(),
			Err:   err,
			Total: &rowIdx,
		}
		return 0
	}
	// 初始化
	queryCache := &QueryData{
		Columns: columns,
		Total:   0,
		Rows:    make([]*QueryDataRow, 0),
	}
	if rowsCount != 1 {
		queryCache.CacheTag = p.CacheHead + crypto.GetMD5(strconv.FormatInt(time.Now().UnixNano(), 10))
	}
	count := len(columns)
	values := make([]interface{}, count)
	scanArgs := make([]interface{}, count)
	for i := range values {
		scanArgs[i] = &values[i]
	}
	// 扫描
	var queryDone bool
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			ch <- &QueryDataChan{
				Data:  newResult(),
				Err:   err,
				Total: &rowIdx,
			}
			return 0
		}
		row := &QueryDataRow{
			VCells: make([]config.VString, count),
			Cells:  make([]string, count),
		}
		for k, v := range values {
			if v == nil {
				row.VCells[k] = ""
				continue
			}
			if b, ok := v.([]uint8); ok {
				row.VCells[k] = config.VString(b)
				row.Cells[k] = gopsu.String(b)
			} else if b, ok := v.(time.Time); ok {
				row.VCells[k] = config.VString(b.Format("2006-01-02 15:04:05"))
				row.Cells[k] = row.VCells[k].String()
			} else {
				row.VCells[k] = config.VString(fmt.Sprintf("%v", v))
				row.Cells[k] = row.VCells[k].String()
			}
		}
		queryCache.Rows = append(queryCache.Rows, row)
		rowIdx++
		if rowsCount > 0 && rowIdx == rowsCount { // 返回
			queryDone = true
			ch <- &QueryDataChan{
				Data: &QueryData{
					Rows:     queryCache.Rows[:rowIdx],
					Total:    queryCache.Total,
					CacheTag: queryCache.CacheTag,
					Columns:  queryCache.Columns},
				Err:   nil,
				Total: &rowIdx,
			}
		}
	}
	queryCache.Total = rowIdx
	if !queryDone { // 全部返回
		ch <- &QueryDataChan{
			Data:  queryCache,
			Err:   nil,
			Total: &rowIdx,
		}
	}
	// 开始缓存，方便导出，有数据即缓存,这里因为已经返回数据，所以不用再开线程
	if p.EnableCache && rowIdx > 0 && rowsCount != 1 { // && rowsCount < rowIdx {
		p.memCache.Store(queryCache.CacheTag, queryCache)
	}
	return rowIdx
}
