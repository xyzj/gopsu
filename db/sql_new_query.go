package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/crypto"
)

// QueryMultirowPage 执行查询语句，返回QueryData结构，检测多个字段进行换行计数
// 如： 采用join关联主表和子表进行查询时，主表的字段会重复，因此应该依据主表字段的变化计算有效记录数据
//
// dbidx: 数据库名称
// s: sql语句
// keyColumeID: 用于分页的关键列id
// rowsCount: 返回数据行数，0-返回全部
// params: 查询参数,对应查询语句中的`？`占位符
func (d *Conn) QueryMultirowPage(dbidx int, s string, rowsCount int, keyColumeID int, params ...interface{}) (query *QueryData, err error) {
	if keyColumeID == -1 {
		return d.Query(s, rowsCount, params...)
	}
	sqldb, err := d.SQLDB(dbidx)
	if err != nil {
		return nil, err
	}
	query = newResult()
	defer func() (*QueryData, error) {
		if ex := recover(); ex != nil {
			err = errors.WithStack(ex.(error))
			return nil, err
		}
		return query, err
	}()
	if rowsCount < 0 {
		rowsCount = 0
	}
	queryCache := newResult()
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	defer cancel()
	rows, err := sqldb.QueryContext(ctx, s, params...)
	if err != nil {
		return query, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return query, err
	}
	queryCache.Columns = columns

	count := len(columns)
	values := make([]interface{}, count)
	scanArgs := make([]interface{}, count)

	for i := range values {
		scanArgs[i] = &values[i]
	}
	queryCache.Rows = make([]*QueryDataRow, 0)
	rowIdx := 0
	limit := 0
	realIdx := 0
	var keyItem string
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			return query, err
		}
		row := &QueryDataRow{
			Cells: make([]string, count),
		}
		for k, v := range values {
			if v == nil {
				row.Cells[k] = ""
			} else {
				b, ok := v.([]byte)
				if ok {
					row.Cells[k] = gopsu.String(b)
				} else {
					row.Cells[k] = fmt.Sprintf("%v", v)
				}
			}
		}
		queryCache.Rows = append(queryCache.Rows, row)
		if keyItem == "" {
			keyItem = row.Cells[keyColumeID]
			rowIdx++
		}
		if keyItem != row.Cells[keyColumeID] {
			keyItem = row.Cells[keyColumeID]
			rowIdx++
		}
		if rowIdx == rowsCount-1 {
			limit = len(queryCache.Rows)
		}
		realIdx++
	}
	if limit == 0 {
		limit = realIdx
	}
	queryCache.Total = rowIdx
	query.Total = queryCache.Total
	query.Columns = queryCache.Columns
	if limit > 0 {
		query.Rows = queryCache.Rows[:limit]
	} else {
		query.Rows = queryCache.Rows
	}

	// 开始缓存，方便导出，有数据即缓存
	if d.cfg.enableCache && rowIdx > 0 { // && rowsCount < rowIdx {
		cacheTag := fmt.Sprintf("%s%d-%d", d.cacheHead, time.Now().UnixNano(), rowIdx)
		query.CacheTag = cacheTag
		queryCache.CacheTag = cacheTag
		go func(qd *QueryData) {
			d.cfg.QueryCache.Store(queryCache.CacheTag, queryCache)
		}(queryCache)
	}
	return query, nil
}

// QueryLimit 执行查询语句，依据startRow和rowsCount自动追加between或limit关键字，用于快速改造原Query方法的结果集
//
// s: sql语句
// startRow: 起始行号，0开始
// rowsCount: 返回数据行数，0-返回全部
// params: 查询参数,对应查询语句中的`？`占位符
func (d *Conn) QueryLimit(s string, startRow, rowsCount int, params ...interface{}) (*QueryData, error) {
	if startRow+rowsCount == 0 {
		return d.Query(s, rowsCount, params...)
	}
	switch d.cfg.DriverType {
	case DriveSQLServer:
		s += fmt.Sprintf(" between %d and %d", startRow, startRow+rowsCount)
	case DriveMySQL:
		s += fmt.Sprintf(" limit %d,%d", startRow, rowsCount)
	}
	query, err := d.Query(s, 0, params...)
	if err != nil {
		return nil, err
	}
	query.CacheTag = emptyCacheTag
	return query, nil
}

// QueryBig 可尝试用于大数据集的首页查询，执行2次查询，第一次查询总数，第二次查询结果集，并立即返回第一页
//
// s: sql语句
// startRow: 起始行号，0开始
// rowsCount: 返回数据行数，0-返回全部
// params: 查询参数,对应查询语句中的`？`占位符
func (d *Conn) QueryBig(dbidx int, s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	sqldb, err := d.SQLDB(dbidx)
	if err != nil {
		return nil, err
	}
	if rowsCount == 0 {
		return d.QueryByDB(dbidx, s, rowsCount, params...)
	}
	ss := "select count(*) " + s[strings.Index(s, "from"):]
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	defer cancel()
	var total int
	err = sqldb.QueryRowContext(ctx, ss, params...).Scan(&total)
	switch {
	case err == sql.ErrNoRows:
		return newResult(), nil
	case err != nil:
		return d.QueryByDB(dbidx, s, rowsCount, params...)
	default:
		qd, err := d.QueryFirstPageByDB(dbidx, s, rowsCount, params...)
		qd.Total = total
		return qd, err
	}
}

// Query 执行查询语句，支持占位符
//
// s： 查询语句
// rowsCount： 需要返回的行数
// params： 参数
func (d *Conn) Query(s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	return d.QueryByDB(d.defaultDB, s, rowsCount, params...)
}

// QueryFirstPage 执行查询语句，返回第一页数据，不返回总数，用于大数据集的首页查询，可通过缓存继续读取后续数据
//
// s： 查询语句
// rowsCount： 需要返回的行数
// params： 参数
func (d *Conn) QueryFirstPage(s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	return d.QueryFirstPageByDB(d.defaultDB, s, rowsCount, params...)
}

// QueryFirstPageByDB 执行查询语句，返回第一页数据，不返回总数，用于大数据集的首页查询，可通过缓存继续读取后续数据，可指定查询的数据库名称
//
// dbidx：执行语句的数据库名称，需要是dbidxs()里面的合法数据库名称
// s： 查询语句
// rowsCount： 需要返回的行数
// params： 参数
func (d *Conn) QueryFirstPageByDB(dbidx int, s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	sqldb, err := d.SQLDB(dbidx)
	if err != nil {
		return nil, err
	}
	if rowsCount == 0 {
		return d.QueryByDB(dbidx, s, rowsCount, params...)
	}
	ch := make(chan *QueryDataChan, 1)
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	go d.queryDataChan(ctx, cancel, sqldb, ch, s, rowsCount, params...)
	select {
	case q := <-ch:
		return q.Data, q.Err
	case <-ctx.Done():
		return newResult(), fmt.Errorf("query data timeout")
	}
}

// QueryByDB 执行查询语句，可指定查询的数据库名称
//
// dbidx：执行语句的数据库名称，需要是dbidxs()里面的合法数据库名称
// s： 查询语句
// rowsCount： 需要返回的行数
// params： 参数
func (d *Conn) QueryByDB(dbidx int, s string, rowsCount int, params ...interface{}) (*QueryData, error) {
	sqldb, err := d.SQLDB(dbidx)
	if err != nil {
		return nil, err
	}
	ch := make(chan *QueryDataChan, 1)
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	qd := newResult()
	go d.queryDataChan(ctx, cancel, sqldb, ch, s, rowsCount, params...)
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

func (d *Conn) queryDataChan(ctx context.Context, done context.CancelFunc, sqldb *sql.DB, ch chan *QueryDataChan, s string, rowsCount int, params ...interface{}) int {
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
	rowIdx := 0
	// 查询数据集
	rows, err := sqldb.QueryContext(ctx, s, params...)
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
		queryCache.CacheTag = d.cacheHead + crypto.GetMD5(strconv.FormatInt(time.Now().UnixNano(), 10))
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
					Columns:  queryCache.Columns,
				},
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
	if d.cfg.enableCache && rowIdx > 0 && rowsCount != 1 { // && rowsCount < rowIdx {
		d.cfg.QueryCache.Store(queryCache.CacheTag, queryCache)
	}
	return rowIdx
}
