/*
Package db : 数据库模块，封装了常用方法，可缓存数据，可依据配置自动创建myisam引擎的子表，支持mysql和sqlserver
*/
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	// ms-sql driver
	_ "github.com/denisenkom/go-mssqldb"
	// mysql driver
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/cache"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/crypto"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
)

var (
	codeGzip = gopsu.GetNewArchiveWorker(gopsu.ArchiveGZip)
)

func qdMarshal(qd *QueryData) ([]byte, error) {
	b, err := json.Marshal(qd)
	if err == nil {
		return codeGzip.Compress(b), nil
	}
	return nil, err
}
func qdUnmarshal(b []byte) *QueryData {
	qd := &QueryData{}
	if err := json.UnmarshalFromString(gopsu.String(codeGzip.Uncompress(b)), qd); err != nil {
		return nil
	}
	return qd
}

// QueryDataRow 数据行
type QueryDataRow struct {
	Cells  []string         `json:"cells,omitempty"`
	VCells []config.VString `json:"vcells,omitempty"`
}

func (d *QueryDataRow) JSON() string {
	s, _ := json.MarshalToString(d)
	return s
}

// QueryData 数据集
type QueryData struct {
	Rows     []*QueryDataRow `json:"rows,omitempty"`
	Columns  []string        `json:"columns,omitempty"`
	CacheTag string          `json:"cache_tag,omitempty"`
	Total    int             `json:"total,omitempty"`
}

func (d *QueryData) JSON() string {
	s, _ := json.MarshalToString(d)
	return s
}

// QueryDataChan chan方式返回首页数据
type QueryDataChan struct {
	Locker *sync.WaitGroup
	Data   *QueryData
	Total  *int
	Err    error
}

// QueryDataChanWorker chan方式数据库访问
type QueryDataChanWorker struct {
	Done        context.CancelFunc
	Params      []interface{}
	QDC         chan *QueryDataChan
	Strsql      string
	RowsCount   int
	keyColumeID int
}

// driveType 数据库驱动类型
type driveType int

const (
	// DriverMYSQL mysql
	DriverMYSQL driveType = iota
	// DriverMSSQL mssql
	DriverMSSQL
)

const (
	emptyCacheTag = "00000-0"
)

func (d driveType) string() string {
	return []string{"mysql", "mssql"}[d]
}

// SQLInterface 数据库接口
type SQLInterface interface {
	New(...string) error
	IsReady() bool
	QueryCacheJSON(string, int, int) string
	QueryCachePB2(string, int, int) *QueryData
	QueryOnePB2(string, int, ...interface{}) (*QueryData, error)
	QueryPB2(string, int, ...interface{}) (*QueryData, error)
	Exec(string, ...interface{}) (int64, int64, error)
	ExecPrepare(string, int, ...interface{}) error
	ExecBatch([]string) error
}

// SQLPool 数据库连接池
type SQLPool struct {
	// 服务地址
	Server string
	// 用户名
	User string
	// 密码
	Passwd string
	// 数据库名称
	DataBase string
	// 数据驱动
	DriverType driveType
	// IO超时(秒)
	Timeout time.Duration
	// 最大连接数
	MaxOpenConns int
	// 日志
	Logger logger.Logger
	// 是否启用缓存功能，缓存30分钟有效
	EnableCache bool
	// 缓存路径
	CacheDir string
	// 缓存文件前缀
	CacheHead string
	// connPool 数据库连接池
	connPool *sql.DB
	// chan方式
	chanQuery chan *QueryDataChanWorker
	// 缓存锁，避免缓存没写完前读取
	cacheLocker sync.Map
	// 内存缓存
	memCache *cache.AnyCache[*QueryData] // *cache.XCache
}

// New 初始化
// tls: 是否启用tls链接。支持以下参数：true,false,skip-verify,preferred
func (p *SQLPool) New(tls ...string) error {
	if p.Server == "" || p.User == "" {
		return errors.New("config error")
	}
	// 处理参数
	if p.Timeout.Seconds() > 6000 || p.Timeout.Seconds() < 5 {
		p.Timeout = time.Second * 300
	}
	if p.MaxOpenConns < 10 {
		p.MaxOpenConns = 10
	}
	if p.MaxOpenConns > 100 {
		p.MaxOpenConns = 100
	}
	if p.CacheDir == "" {
		p.CacheDir = gopsu.DefaultCacheDir
	}
	if p.Logger == nil {
		p.Logger = &logger.NilLogger{}
	}
	// 设置参数
	var connstr string
	switch p.DriverType {
	case DriverMSSQL:
		connstr = fmt.Sprintf("user id=%s;"+
			"password=%s;"+
			"server=%s;"+
			"database=%s;"+
			"connection timeout=180",
			p.User, p.Passwd, p.Server, p.DataBase)
		if len(tls) > 0 {
			if tls[0] != "false" {
				connstr += ";encrypt=true;trustservercertificate=true"
			}
		}
	case DriverMYSQL:
		sqlcfg := &mysql.Config{
			Collation:            "utf8_general_ci",
			Loc:                  time.Local,
			MaxAllowedPacket:     0, // 64*1024*1024
			AllowNativePasswords: true,
			CheckConnLiveness:    true,
			Net:                  "tcp",
			Addr:                 p.Server,
			User:                 p.User,
			Passwd:               p.Passwd,
			DBName:               p.DataBase,
			MultiStatements:      true,
			ParseTime:            true,
			Timeout:              time.Second * 180,
			ColumnsWithAlias:     true,
			ClientFoundRows:      true,
			InterpolateParams:    true,
		}
		if len(tls) > 0 {
			sqlcfg.TLSConfig = tls[0]
		}
		connstr = sqlcfg.FormatDSN()
	}

	if p.CacheHead == "" {
		p.CacheHead = gopsu.CalcCRC32String([]byte(connstr))
	}
	p.memCache = cache.NewAnyCache[*QueryData](time.Hour) // cache.NewCacheWithWriter(0, p.Logger.DefaultWriter())
	// 连接/测试
	db, err := sql.Open(p.DriverType.string(), strings.ReplaceAll(connstr, "\n", ""))
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(p.MaxOpenConns)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	p.connPool = db

	// 缓存文件维护
	// if p.EnableCache {
	// 	go func() {
	// 		defer func() {
	// 			if err := recover(); err != nil {
	// 				p.Logger.Error("[DB] SQL Cache file clean error:" + errors.WithStack(err.(error)).Error())
	// 			}
	// 		}()
	// 		t := time.NewTicker(time.Minute * 45)
	// 		for range t.C {
	// 			// 维护缓存文件
	// 			p.checkCache()
	// 		}
	// 	}()
	// }
	// 通道访问,并发数量限制在连接池的一半
	// p.chanQuery = make(chan *QueryDataChanWorker, p.MaxOpenConns)
	// for i := 0; i < p.MaxOpenConns/2; i++ {
	// 	go func() {
	// 		locker := &sync.WaitGroup{}
	// 	CREATE:
	// 		locker.Add(1)
	// 		go func() {
	// 			defer func() {
	// 				if err := recover(); err != nil {
	// 					p.Logger.Error("[DB] SQL channel worker error:" + errors.WithStack(err.(error)).Error())
	// 				}
	// 				locker.Done()
	// 			}()
	// 			for cq := range p.chanQuery {
	// 				// 调用chan方法
	// 				p.queryChan(cq.QDC, cq.Strsql, cq.RowsCount, cq.Params...)
	// 				// p.queryDataChan(cq.Done, cq.QDC, cq.Strsql, cq.RowsCount, cq.Params...)
	// 			}
	// 		}()
	// 		locker.Wait()
	// 		goto CREATE
	// 	}()
	// }
	// 启动结束
	p.Logger.System("[DB] Success connect to server " + p.Server)
	return nil
}

// IsReady 检查状态
func (p *SQLPool) IsReady() bool {
	if p.connPool == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := p.connPool.PingContext(ctx); err != nil {
		return false
	}
	return true

}

// checkSQL 检查sql语句是否存在注入攻击风险
//
// args：
//
// s： sql语句
func (p *SQLPool) checkSQL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if gopsu.CheckSQLInject(s) {
		return nil
	}
	return errors.New("SQL statement has risk of injection: " + s)
}

// 维护缓存文件数量
func (p *SQLPool) checkCache() {
	files, err := os.ReadDir(p.CacheDir)
	if err != nil {
		return
	}
	t := time.Now()
	for _, d := range files {
		if d.IsDir() {
			continue
		}
		file, err := d.Info()
		if err != nil {
			continue
		}
		if !strings.HasPrefix(file.Name(), p.CacheHead) {
			continue
		}
		// 删除文件
		if t.Sub(file.ModTime()).Minutes() > 30 {
			os.Remove(filepath.Join(p.CacheDir, file.Name()))
		}
		// 整理
	}
}

// QueryCacheJSON 查询缓存结果，返回json字符串
//
// cacheTag: 缓存标签
//
// startIdx: 起始行数
//
// rowCount: 查询的行数
func (p *SQLPool) QueryCacheJSON(cacheTag string, startRow, rowsCount int) string {
	// return gopsu.String(gopsu.PB2Json(p.QueryCachePB2(cacheTag, startRow, rowsCount)))
	return p.QueryCachePB2(cacheTag, startRow, rowsCount).JSON()
}

// QueryCachePB2 查询缓存结果，返回QueryData结构
//
// cacheTag: 缓存标签
//
// startIdx: 起始行数
//
// rowCount: 查询的行数
func (p *SQLPool) QueryCachePB2(cacheTag string, startRow, rowsCount int) *QueryData {
	if cacheTag == emptyCacheTag {
		return nil
	}
	if startRow < 1 {
		startRow = 1
	}
	if rowsCount < 0 {
		rowsCount = 0
	}
	query := &QueryData{
		CacheTag: cacheTag,
		Rows:     make([]*QueryDataRow, 0),
	}
	// 读取前等待写入完毕,使用memcache不需要等
	// if lo, ok := p.cacheLocker.Load(cacheTag); ok {
	// 	lo.(*sync.WaitGroup).Wait()
	// }
	// 开始读取
	if src, ok := p.memCache.Load(cacheTag); ok {
		if msg := src; msg != nil {
			// if src, err := os.ReadFile(filepath.Join(p.CacheDir, cacheTag)); err == nil {
			// if msg := qdUnmarshal(src); msg != nil {
			query.Total = msg.Total
			startRow = startRow - 1
			endRow := startRow + rowsCount
			if rowsCount == 0 || endRow > len(msg.Rows) {
				endRow = int(msg.Total)
			}
			if startRow >= int(msg.Total) {
				query.Total = 0
			} else {
				query.Total = msg.Total
				query.Rows = msg.Rows[startRow:endRow]
			}
		}
	}
	return query
}

// QueryCacheMultirowPage 查询多行分页缓存结果，返回QueryData结构
//
// cacheTag: 缓存标签
//
// startIdx: 起始行数
//
// rowCount: 查询的行数
func (p *SQLPool) QueryCacheMultirowPage(cacheTag string, startRow, rowsCount, keyColumeID int) *QueryData {
	if cacheTag == emptyCacheTag {
		return nil
	}
	if keyColumeID == -1 {
		return p.QueryCachePB2(cacheTag, startRow, rowsCount)
	}
	if startRow < 1 {
		startRow = 1
	}
	if rowsCount < 0 {
		rowsCount = 0
	}
	query := &QueryData{CacheTag: cacheTag}
	if src, ok := p.memCache.Load(cacheTag); ok {
		if msg := src; msg != nil {
			// if src, err := os.ReadFile(filepath.Join(p.CacheDir, cacheTag)); err == nil {
			// if msg := qdUnmarshal(src); msg != nil {
			startRow = startRow - 1
			query.Total = msg.Total
			endRow := startRow + rowsCount
			if rowsCount == 0 {
				endRow = int(msg.Total)
			}
			if startRow >= int(msg.Total) {
				query.Total = 0
			} else {
				query.Total = msg.Total
				var rowIdx int
				var keyItem string
				for _, v := range msg.Rows {
					if keyItem == "" {
						keyItem = v.Cells[keyColumeID]
					}
					if keyItem != v.Cells[keyColumeID] {
						keyItem = v.Cells[keyColumeID]
						rowIdx++
					}
					if rowIdx >= startRow && rowIdx < endRow {
						query.Rows = append(query.Rows, v)
					}
				}
			}
		}
	}
	return query
}

// QueryOne 执行查询语句，返回首行结果的json字符串，`{row：[...]}`，该方法不缓存结果
//
// Deprecated: use Query() or QueryJSON()
func (p *SQLPool) QueryOne(s string, colNum int, params ...interface{}) (js string, err error) {
	pb, err := p.QueryFirstPage(s, 1, params...)
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
// Deprecated: use QueryFirstPage()
func (p *SQLPool) QueryOnePB2(s string, colNum int, params ...interface{}) (query *QueryData, err error) {
	qd, err := p.QueryFirstPage(s, 1, params...)
	if err != nil {
		return qd, err
	}
	qd.Total = len(qd.Rows)
	return qd, nil

	// query = &QueryData{Rows: make([]*QueryDataRow, 0)}
	// defer func() (*QueryData, error) {
	// 	if ex := recover(); ex != nil {
	// 		err = errors.WithStack(ex.(error))
	// 		return nil, err
	// 	}
	// 	return query, nil
	// }()

	// values := make([]interface{}, colNum)
	// scanArgs := make([]interface{}, colNum)

	// for i := range values {
	// 	scanArgs[i] = &values[i]
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	// defer cancel()
	// err = p.connPool.QueryRowContext(ctx, s, params...).Scan(scanArgs...)
	// switch {
	// case err == sql.ErrNoRows:
	// 	return query, nil
	// case err != nil:
	// 	return nil, err
	// default:
	// 	query.Total = 1
	// 	query.Rows = []*QueryDataRow{{Cells: make([]string, colNum)}}
	// 	for i := range scanArgs {
	// 		v := values[i]
	// 		b, ok := v.([]byte)
	// 		if ok {
	// 			query.Rows[0].Cells[i] = gopsu.String(b)
	// 			// query.Rows[0].Cells = append(query.Rows[0].Cells, gopsu.String(b))
	// 		} else {
	// 			query.Rows[0].Cells[i] = fmt.Sprintf("%v", v)
	// 			// query.Rows[0].Cells = append(query.Rows[0].Cells, fmt.Sprintf("%v", v))
	// 		}
	// 	}
	// 	return query, nil
	// }
}

// QueryPB2 执行查询语句，返回QueryData结构
//
// Deprecated: use Query()
func (p *SQLPool) QueryPB2(s string, rowsCount int, params ...interface{}) (query *QueryData, err error) {
	return p.Query(s, rowsCount, params...)
	// ans := <-p.QueryPB2Chan(s, rowsCount, params...)
	// if ans.Err != nil {
	// 	return nil, ans.Err
	// }
	// if ans.Locker != nil {
	// 	ans.Locker.Wait()
	// }
	// ans.Data.Total = *ans.Total
	// return ans.Data, nil
}

// QueryPB2Chan 查询v2,采用线程+channel优化超大数据集分页的首页返回时间
//
// Deprecated: use Query() or QueryFirstPage() or QueryBig()
func (p *SQLPool) QueryPB2Chan(s string, rowsCount int, params ...interface{}) <-chan *QueryDataChan {
	var ch = make(chan *QueryDataChan, 1)
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	go p.queryDataChan(ctx, cancel, ch, s, rowsCount, params...)
	// go p.queryChan(ch, s, rowsCount, params...)
	return ch
	// qdc := &QueryDataChanWorker{
	// 	QDC:         make(chan *QueryDataChan, 1),
	// 	RowsCount:   rowsCount,
	// 	Strsql:      s,
	// 	Params:      params,
	// 	keyColumeID: -1,
	// }
	// p.chanQuery <- qdc
	// return qdc.QDC
}
func (p *SQLPool) queryChan(qdc chan *QueryDataChan, s string, rowsCount int, params ...interface{}) {
	defer func() {
		if err := recover(); err != nil {
			qdc <- &QueryDataChan{
				Data: nil,
				Err:  errors.WithStack(err.(error)),
			}
		}
	}()

	if rowsCount < 0 {
		rowsCount = 0
	}
	// 查询总行数
	var total int
	// 查询数据集
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()
	rows, err := p.connPool.QueryContext(ctx, s, params...)
	if err != nil {
		qdc <- &QueryDataChan{
			Data: nil,
			Err:  err,
		}
		return
	}
	defer rows.Close()
	// 处理数据集
	columns, err := rows.Columns()
	if err != nil {
		qdc <- &QueryDataChan{
			Data: nil,
			Err:  err,
		}
		return
	}
	// 初始化
	queryCache := &QueryData{
		Columns:  columns,
		Total:    0,
		Rows:     make([]*QueryDataRow, 0),
		CacheTag: p.CacheHead + crypto.GetMD5(strconv.FormatInt(time.Now().UnixNano(), 10)),
	}
	count := len(columns)
	values := make([]interface{}, count)
	scanArgs := make([]interface{}, count)

	for i := range values {
		scanArgs[i] = &values[i]
	}
	// 扫描
	var queryDone bool
	var rowIdx = 0
	var totalLocker = &sync.WaitGroup{}
	totalLocker.Add(1)
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			qdc <- &QueryDataChan{
				Data: queryCache,
				Err:  err,
			}
			totalLocker.Done()
			return
		}
		row := &QueryDataRow{
			Cells:  make([]string, count),
			VCells: make([]config.VString, count),
		}
		for k, v := range values {
			if v == nil {
				row.Cells[k] = ""
				row.VCells[k] = ""
				continue
			}
			if b, ok := v.([]uint8); ok {
				row.Cells[k] = gopsu.String(b)
				// row.VCells[k] = config.VString(b)
			} else if b, ok := v.(time.Time); ok {
				row.Cells[k] = b.Format("2006-01-02 15:04:05")
				// row.VCells[k] = config.VString(b.Format("2006-01-02 15:04:05"))
			} else {
				row.Cells[k] = fmt.Sprintf("%v", v)
				// row.VCells[k] = config.VString(fmt.Sprintf("%v", v))
			}
		}
		// queryCache.Rows[rowIdx] = row
		queryCache.Rows = append(queryCache.Rows, row)
		rowIdx++
		if rowsCount > 0 && rowIdx == rowsCount { // 返回
			queryDone = true
			qdc <- &QueryDataChan{
				Data: &QueryData{
					Rows:     queryCache.Rows[:rowIdx],
					Total:    queryCache.Total,
					CacheTag: queryCache.CacheTag,
					Columns:  queryCache.Columns,
				},
				Err:    nil,
				Total:  &total,
				Locker: totalLocker,
			}
		}
	}
	queryCache.Total = rowIdx
	total = rowIdx
	totalLocker.Done()
	if !queryDone { // 全部返回
		qdc <- &QueryDataChan{
			Data:   queryCache,
			Err:    nil,
			Total:  &total,
			Locker: totalLocker,
		}
	}
	// 开始缓存，方便导出，有数据即缓存,这里因为已经返回数据，所以不用再开线程
	if p.EnableCache && rowIdx > 0 { // && rowsCount < rowIdx {
		p.memCache.Store(queryCache.CacheTag, queryCache)
		// lo := &sync.WaitGroup{}
		// lo.Add(1)
		// p.cacheLocker.Store(queryCache.CacheTag, lo)
		// if b, err := qdMarshal(queryCache); err == nil {
		// 	os.WriteFile(filepath.Join(gopsu.DefaultCacheDir, queryCache.CacheTag), b, 0664)
		// }
		// lo.Done()
		// p.cacheLocker.Delete(queryCache.CacheTag)
	}
}

// QueryMultirowPage 执行查询语句，返回QueryData结构，检测多个字段进行换行计数
//
// s: sql语句
//
// keyColumeID: 用于分页的关键列id
//
// rowsCount: 返回数据行数，0-返回全部
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryMultirowPage(s string, rowsCount int, keyColumeID int, params ...interface{}) (query *QueryData, err error) {
	if keyColumeID == -1 {
		return p.Query(s, rowsCount, params...)
	}
	query = &QueryData{}
	// p.queryLocker.Lock()
	defer func() (*QueryData, error) {
		if ex := recover(); ex != nil {
			err = errors.WithStack(ex.(error))
			return nil, err
		}
		// p.queryLocker.Unlock()
		return query, err
	}()
	if rowsCount < 0 {
		rowsCount = 0
	}
	queryCache := &QueryData{}
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()
	rows, err := p.connPool.QueryContext(ctx, s, params...)
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
	var rowIdx = 0
	var limit = 0
	var realIdx = 0
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
	// rowIdx++
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
	if p.EnableCache && rowIdx > 0 { // && rowsCount < rowIdx {
		cacheTag := fmt.Sprintf("%s%d-%d", p.CacheHead, time.Now().UnixNano(), rowIdx)
		query.CacheTag = cacheTag
		queryCache.CacheTag = cacheTag
		go func(qd *QueryData) {
			p.memCache.Store(queryCache.CacheTag, queryCache)
			// lo := &sync.WaitGroup{}
			// lo.Add(1)
			// p.cacheLocker.Store(queryCache.CacheTag, lo)
			// if b, err := qdMarshal(queryCache); err == nil {
			// 	os.WriteFile(filepath.Join(p.CacheDir, cacheTag), b, 0664)
			// }
			// lo.Done()
			// p.cacheLocker.Delete(queryCache.CacheTag)
		}(queryCache)
	}
	return query, nil
}
