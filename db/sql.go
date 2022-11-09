/*
Package db : 数据库模块，封装了常用方法，可缓存数据，可依据配置自动创建myisam引擎的子表，支持mysql和sqlserver
*/
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
)

var (
	codeGzip    = gopsu.GetNewArchiveWorker(gopsu.ArchiveGZip)
	cacheWroker = gopsu.GetNewCryptoWorker(gopsu.CryptoMD5)
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
	Cells []string `json:"cells,omitempty"`
}

// QueryData 数据集
type QueryData struct {
	Total    int32           `json:"total,omitempty"`
	CacheTag string          `json:"cache_tag,omitempty"`
	Rows     []*QueryDataRow `json:"rows,omitempty"`
	Columns  []string        `json:"columns,omitempty"`
}

// QueryDataChan chan方式返回首页数据
type QueryDataChan struct {
	Data   *QueryData
	Err    error
	Total  *int
	Locker *sync.WaitGroup
}

// QueryDataChanWorker chan方式数据库访问
type QueryDataChanWorker struct {
	QDC         chan *QueryDataChan
	RowsCount   int
	Strsql      string
	Params      []interface{}
	keyColumeID int
}

// driveType 数据库驱动类型
type driveType int

const (
	// DriverMYSQL mysql
	DriverMYSQL driveType = iota
	// DriverMSSQL mssql
	DriverMSSQL
	// DriverOracle oracle
	DriverOracle
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
	Timeout int
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
	// 查询锁
	queryLocker sync.Mutex
	execLocker  sync.Mutex
	// chan方式
	chanQuery chan *QueryDataChanWorker
	// 缓存锁，避免缓存没写完前读取
	cacheLocker sync.Map
	// 内存缓存
	memCache *cache.XCache
}

// New 初始化
// tls: 是否启用tls链接。支持以下参数：true,false,skip-verify,preferred
func (p *SQLPool) New(tls ...string) error {
	if p.Server == "" || p.User == "" || p.Passwd == "" {
		return errors.New("config error")
	}
	// 处理参数
	if p.Timeout > 6000 || p.Timeout < 5 {
		p.Timeout = 120
	}
	if p.MaxOpenConns < 20 {
		p.MaxOpenConns = 20
	}
	if p.MaxOpenConns > 500 {
		p.MaxOpenConns = 500
	}
	if p.CacheDir == "" {
		p.CacheDir = gopsu.DefaultCacheDir
	}
	if p.Logger == nil {
		p.Logger = &logger.NilLogger{}
	}
	// p.cacheLocker = make(map[string]*sync.WaitGroup)
	// 设置参数
	var connstr string
	switch p.DriverType {
	case DriverMSSQL:
		connstr = fmt.Sprintf("user id=%s;"+
			"password=%s;"+
			"server=%s;"+
			"database=%s;"+
			"connection timeout=10",
			p.User, p.Passwd, p.Server, p.DataBase)
		if len(tls) > 0 {
			if tls[0] != "false" {
				connstr += ";encrypt=true;trustservercertificate=true"
			}
		}
	case DriverMYSQL:
		sqlcfg := &mysql.Config{
			Collation:            "utf8_general_ci",
			Loc:                  time.UTC,
			MaxAllowedPacket:     4 << 20,
			AllowNativePasswords: true,
			CheckConnLiveness:    true,
			Net:                  "tcp",
			Addr:                 p.Server,
			User:                 p.User,
			Passwd:               p.Passwd,
			DBName:               p.DataBase,
			MultiStatements:      true,
			ParseTime:            true,
			Timeout:              time.Second * 10,
			ColumnsWithAlias:     true,
			ClientFoundRows:      true,
			InterpolateParams:    true,
		}
		if len(tls) > 0 {
			sqlcfg.TLSConfig = tls[0]
		}
		connstr = sqlcfg.FormatDSN()
		// connstr = fmt.Sprintf("%s:%s@tcp(%s)/%s"+
		// 	"?multiStatements=true"+
		// 	"&parseTime=true"+
		// 	"&timeout=10s"+
		// 	"&charset=utf8"+
		// 	"&columnsWithAlias=true"+
		// 	"&clientFoundRows=true",
		// 	p.User, p.Passwd, p.Server, p.DataBase)
	}

	if p.CacheHead == "" {
		p.CacheHead = gopsu.CalcCRC32String([]byte(connstr))
	}
	p.memCache = cache.NewCache(0, p.Logger.DefaultWriter())
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
	if p.EnableCache {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					p.Logger.Error("SQL Cache file clean error:" + errors.WithStack(err.(error)).Error())
				}
			}()
			t := time.NewTicker(time.Minute * 45)
			for range t.C {
				// 维护缓存文件
				p.checkCache()
			}
		}()
	}
	// 通道访问,并发数量限制在连接池的一半
	p.chanQuery = make(chan *QueryDataChanWorker, p.MaxOpenConns)
	for i := 0; i < p.MaxOpenConns/2; i++ {
		go func() {
			locker := &sync.WaitGroup{}
		CREATE:
			locker.Add(1)
			go func() {
				defer func() {
					if err := recover(); err != nil {
						p.Logger.Error("SQL channel worker error:" + errors.WithStack(err.(error)).Error())
					}
					locker.Done()
				}()
				for cq := range p.chanQuery {
					// 调用chan方法
					p.queryChan(cq.QDC, cq.Strsql, cq.RowsCount, cq.Params...)
				}
			}()
			locker.Wait()
			goto CREATE
		}()
	}
	// 启动结束
	p.Logger.System("Success connect to server " + p.Server)
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
	if gopsu.CheckSQLInject(s) {
		return nil
	}
	return errors.New("SQL statement has risk of injection")
}

// 维护缓存文件数量
func (p *SQLPool) checkCache() {
	files, err := ioutil.ReadDir(p.CacheDir)
	if err != nil {
		return
	}
	t := time.Now()
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), p.CacheHead) {
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
	return gopsu.String(gopsu.PB2Json(p.QueryCachePB2(cacheTag, startRow, rowsCount)))
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
	if src, ok := p.memCache.GetAndExpire(cacheTag, time.Minute*30); ok {
		if msg := src.(*QueryData); msg != nil {
			// if src, err := ioutil.ReadFile(filepath.Join(p.CacheDir, cacheTag)); err == nil {
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
	if src, ok := p.memCache.GetAndExpire(cacheTag, time.Minute*30); ok {
		if msg := src.(*QueryData); msg != nil {
			// if src, err := ioutil.ReadFile(filepath.Join(p.CacheDir, cacheTag)); err == nil {
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
// s: sql语句
//
// colNum: 列数量
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryOne(s string, colNum int, params ...interface{}) (js string, err error) {
	pb, err := p.QueryPB2(s, colNum, params...)
	if err != nil {
		return "", err
	}
	if pb.Total == 0 {
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
// s: sql语句
//
// colNum: 列数量
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryOnePB2(s string, colNum int, params ...interface{}) (query *QueryData, err error) {
	query = &QueryData{Rows: make([]*QueryDataRow, 0)}
	defer func() (*QueryData, error) {
		if ex := recover(); ex != nil {
			err = errors.WithStack(ex.(error))
			return nil, err
		}
		return query, nil
	}()

	values := make([]interface{}, colNum)
	scanArgs := make([]interface{}, colNum)

	for i := range values {
		scanArgs[i] = &values[i]
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	defer cancel()
	err = p.connPool.QueryRowContext(ctx, s, params...).Scan(scanArgs...)
	switch {
	case err == sql.ErrNoRows:
		return query, nil
	case err != nil:
		return nil, err
	default:
		query.Total = 1
		query.Rows = []*QueryDataRow{&QueryDataRow{Cells: make([]string, colNum)}}
		for i := range scanArgs {
			v := values[i]
			b, ok := v.([]byte)
			if ok {
				query.Rows[0].Cells[i] = gopsu.String(b)
				// query.Rows[0].Cells = append(query.Rows[0].Cells, gopsu.String(b))
			} else {
				query.Rows[0].Cells[i] = fmt.Sprintf("%v", v)
				// query.Rows[0].Cells = append(query.Rows[0].Cells, fmt.Sprintf("%v", v))
			}
		}
		return query, nil
	}
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
		return p.QueryPB2(s, rowsCount, params...)
	}
	switch p.DriverType {
	case DriverMSSQL:
		s += fmt.Sprintf(" between %d and %d", startRow, startRow+rowsCount)
	case DriverMYSQL:
		s += fmt.Sprintf(" limit %d,%d", startRow, rowsCount)
	}
	query, err := p.QueryPB2(s, 0, params...)
	if err != nil {
		return nil, err
	}
	query.CacheTag = emptyCacheTag
	return query, nil
}

// QueryPB2Big 可尝试用于大数据集的首页查询，一定程度加快速度，原查询时间在2s内的没必要使用该方法
//
// s: sql语句
//
// startRow: 起始行号，0开始
//
// rowsCount: 返回数据行数，0-返回全部
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryPB2Big(s string, startRow, rowsCount int, params ...interface{}) (*QueryData, error) {
	// ss := strings.Replace(s, "select ", "select count(*),", 1)
	ss := "select count(*) " + s[strings.Index(s, "from"):]
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	defer cancel()
	var total int
	err := p.connPool.QueryRowContext(ctx, ss, params...).Scan(&total)
	switch {
	case err == sql.ErrNoRows:
		return &QueryData{
			Total: 0,
		}, nil
	case err != nil:
		return p.QueryPB2(s, rowsCount, params...)
	default:
		query, err := p.QueryLimit(s, startRow, rowsCount, params...)
		query.Total = int32(total)
		return query, err
	}
}

// QueryJSON 执行查询语句，返回结果集的json字符串
//
// s: sql语句
//
// rowsCount: 返回数据行数，0-返回全部
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryJSON(s string, rowsCount int, params ...interface{}) (string, error) {
	x, ex := p.QueryPB2(s, rowsCount, params...)
	if ex != nil {
		return "", ex
	}
	return gopsu.String(gopsu.PB2Json(x)), nil
}

// QueryPB2 执行查询语句，返回QueryData结构
//
// s: sql语句
//
// rowsCount: 返回数据行数，0-返回全部
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryPB2(s string, rowsCount int, params ...interface{}) (query *QueryData, err error) {
	ans := <-p.QueryPB2Chan(s, rowsCount, params...)
	if ans.Err != nil {
		return nil, ans.Err
	}
	if ans.Locker != nil {
		ans.Locker.Wait()
	}
	ans.Data.Total = int32(*ans.Total)
	return ans.Data, nil

	// query = &QueryData{Rows: make([]*QueryDataRow, 0)}
	// defer func() (*QueryData, error) {
	// 	if ex := recover(); ex != nil {
	// 		err = errors.WithStack(err.(error))
	// 		return nil, err
	// 	}
	// 	return query, err
	// }()

	// if rowsCount < 0 {
	// 	rowsCount = 0
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	// defer cancel()
	// rows, err := p.connPool.QueryContext(ctx, s, params...)
	// if err != nil {
	// 	return nil, err
	// }
	// defer rows.Close()
	// columns, err := rows.Columns()
	// if err != nil {
	// 	return nil, err
	// }
	// queryCache := &QueryData{
	// 	Columns: columns,
	// }

	// count := len(columns)
	// values := make([]interface{}, count)
	// scanArgs := make([]interface{}, count)
	// for i := range values {
	// 	scanArgs[i] = &values[i]
	// }
	// // 开始遍历结果集
	// var queryDone bool
	// queryCache.Rows = make([]*QueryDataRow, 0)
	// var rowIdx = 0
	// for rows.Next() {
	// 	err := rows.Scan(scanArgs...)
	// 	if err != nil {
	// 		return queryCache, err
	// 	}
	// 	row := &QueryDataRow{
	// 		Cells: make([]string, count),
	// 	}
	// 	for k, v := range values {
	// 		if v == nil {
	// 			row.Cells[k] = ""
	// 			continue
	// 		}
	// 		if b, ok := v.([]byte); ok {
	// 			row.Cells[k] = gopsu.String(b)
	// 		} else {
	// 			row.Cells[k] = fmt.Sprintf("%v", v)
	// 		}
	// 	}
	// 	queryCache.Rows = append(queryCache.Rows, row)
	// 	rowIdx++
	// 	if rowsCount > 0 && rowIdx == rowsCount {
	// 		queryDone = true
	// 		query.Rows = queryCache.Rows[:rowIdx]
	// 	}
	// }
	// queryCache.Total = int32(rowIdx)
	// query.Total = int32(rowIdx)
	// query.Columns = queryCache.Columns
	// if !queryDone {
	// 	query.Rows = queryCache.Rows[:rowIdx]
	// }
	// // 开始缓存，方便导出，有数据即缓存
	// if p.EnableCache && rowIdx > 0 { // && rowsCount < rowIdx {
	// 	cacheTag := fmt.Sprintf("%s%s-%d", p.CacheHead, cacheWroker.Hash(gopsu.Bytes(fmt.Sprintf("%d", time.Now().UnixNano()))), rowIdx)
	// 	query.CacheTag = cacheTag
	// 	queryCache.CacheTag = cacheTag
	// 	go func(qd *QueryData) {
	// 		lo := &sync.WaitGroup{}
	// 		lo.Add(1)
	// 		p.cacheLocker.Store(qd.CacheTag, lo)
	// 		if b, err := qdMarshal(qd); err == nil {
	// 			ioutil.WriteFile(filepath.Join(p.CacheDir, qd.CacheTag), b, 0664)
	// 		}
	// 		lo.Done()
	// 		p.cacheLocker.Delete(qd.CacheTag)
	// 	}(queryCache)
	// }
	// return query, nil
}

// QueryPB2Chan 查询v2,采用线程+channel优化超大数据集分页的首页返回时间
//
// s: sql语句
//
// rowsCount: 返回数据行数，0-返回全部
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) QueryPB2Chan(s string, rowsCount int, params ...interface{}) <-chan *QueryDataChan {
	qdc := &QueryDataChanWorker{
		QDC:         make(chan *QueryDataChan, 1),
		RowsCount:   rowsCount,
		Strsql:      s,
		Params:      params,
		keyColumeID: -1,
	}
	p.chanQuery <- qdc
	return qdc.QDC
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
	// ss := strings.Replace(s, "select ", "select SQL_CALC_FOUND_ROWS ", 1) + "; select FOUND_ROWS();"
	// ss := "select count(*) from (" + s+") T"
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	// defer cancel()
	// err := p.connPool.QueryRowContext(ctx, ss, params...).Scan(&total)
	// switch {
	// case err != nil:
	// 	p.Logger.Error("QueryPB2Chan Err: " + err.Error())
	// 	qdc <- &QueryDataChan{
	// 		Data: nil,
	// 		Err:  err,
	// 	}
	// 	return
	// default:
	// 	if total == 0 {
	// 		qdc <- &QueryDataChan{
	// 			Data: &QueryData{},
	// 			Err:  nil,
	// 		}
	// 		return
	// 	}
	// }
	// 查询数据集
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
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
		CacheTag: p.CacheHead + cacheWroker.Hash(gopsu.Bytes(fmt.Sprintf("%d", time.Now().UnixNano()))),
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
			Cells: make([]string, count),
		}
		for k, v := range values {
			if v == nil {
				row.Cells[k] = ""
				continue
			}
			if b, ok := v.([]byte); ok {
				row.Cells[k] = gopsu.String(b)
			} else {
				row.Cells[k] = fmt.Sprintf("%v", v)
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
	queryCache.Total = int32(rowIdx)
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
		p.memCache.Set(queryCache.CacheTag, queryCache, time.Minute*30)
		// lo := &sync.WaitGroup{}
		// lo.Add(1)
		// p.cacheLocker.Store(queryCache.CacheTag, lo)
		// if b, err := qdMarshal(queryCache); err == nil {
		// 	ioutil.WriteFile(filepath.Join(gopsu.DefaultCacheDir, queryCache.CacheTag), b, 0664)
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
		return p.QueryPB2(s, rowsCount, params...)
	}
	query = &QueryData{}
	// p.queryLocker.Lock()
	defer func() (*QueryData, error) {
		if ex := recover(); ex != nil {
			err = errors.WithStack(err.(error))
			return nil, err
		}
		// p.queryLocker.Unlock()
		return query, err
	}()
	if rowsCount < 0 {
		rowsCount = 0
	}
	queryCache := &QueryData{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
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
			// rowIdx++
		}
		if keyItem != row.Cells[keyColumeID] {
			keyItem = row.Cells[keyColumeID]
			rowIdx++
		}
	}
	rowIdx++
	if rowsCount > rowIdx {
		rowsCount = rowIdx
	}
	queryCache.Total = int32(rowIdx)
	query.Total = queryCache.Total
	query.Columns = queryCache.Columns
	if rowsCount > 0 {
		query.Rows = queryCache.Rows[:rowsCount]
	} else {
		query.Rows = queryCache.Rows
	}

	// 开始缓存，方便导出，有数据即缓存
	if p.EnableCache && rowIdx > 0 { // && rowsCount < rowIdx {
		cacheTag := fmt.Sprintf("%s%d-%d", p.CacheHead, time.Now().UnixNano(), rowIdx)
		query.CacheTag = cacheTag
		queryCache.CacheTag = cacheTag
		go func(qd *QueryData) {
			p.memCache.Set(queryCache.CacheTag, queryCache, time.Minute*30)
			// lo := &sync.WaitGroup{}
			// lo.Add(1)
			// p.cacheLocker.Store(queryCache.CacheTag, lo)
			// if b, err := qdMarshal(queryCache); err == nil {
			// 	ioutil.WriteFile(filepath.Join(p.CacheDir, cacheTag), b, 0664)
			// }
			// lo.Done()
			// p.cacheLocker.Delete(queryCache.CacheTag)
		}(queryCache)
	}
	return query, nil
}

// Exec 执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用官方的语句参数分离写法
//
// s: sql语句
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) Exec(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	p.execLocker.Lock()
	defer func() (int64, int64, error) {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return 0, 0, err
		}
		p.execLocker.Unlock()
		return rowAffected, insertID, nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	defer cancel()
	res, err := p.connPool.ExecContext(ctx, s, params...)
	if err != nil {
		return 0, 0, err
	}
	insertID, _ = res.LastInsertId()
	rowAffected, _ = res.RowsAffected()
	return rowAffected, insertID, nil
}

// ExecV2 事务执行语句（insert，delete，update），可回滚,返回（影响行数,insertId,error）,使用官方的语句参数分离写法
//
// s: sql语句
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecV2(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	// p.execLocker.Lock()
	defer func() (int64, int64, error) {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return 0, 0, err
		}
		// p.execLocker.Unlock()
		return rowAffected, insertID, nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	defer cancel()
	// 开启事务
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			p.Logger.Error(err.Error())
		}
	}()
	res, err := tx.ExecContext(ctx, s, params...)
	if err != nil {
		return 0, 0, err
	}
	err = tx.Commit()
	if err != nil {
		// tx.Rollback()
		return 0, 0, err
	}
	// res, err := p.connPool.ExecContext(ctx, s, params...)
	// if err != nil {
	// 	return 0, 0, err
	// }
	insertID, _ = res.LastInsertId()
	rowAffected, _ = res.RowsAffected()
	return rowAffected, insertID, nil
}

// ExecPrepare 批量执行占位符语句 返回 err，使用官方的语句参数分离写法，用于批量执行相同语句
//
// s: sql语句
//
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecPrepare(s string, paramNum int, params ...interface{}) (err error) {
	p.execLocker.Lock()
	defer func() error {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return err
		}
		p.execLocker.Unlock()
		return nil
	}()
	if paramNum == 0 {
		paramNum = strings.Count(s, "?")
	}

	l := len(params)
	if l%paramNum != 0 {
		return errors.New("not enough params")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	defer cancel()
	// 开启事务
	// st, err := p.connPool.PrepareContext(ctx, s)
	// if err != nil {
	// 	return err
	// }
	// defer st.Close()
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			p.Logger.Error(err.Error())
		}
	}()
	for i := 0; i < l; i += paramNum {
		_, err := tx.ExecContext(ctx, s, params[i:i+paramNum]...)
		// _, err = tx.StmtContext(ctx, st).Exec(params[i : i+paramNum]...)
		// _, err := st.ExecContext(ctx, params[i:i+paramNum]...)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		// tx.Rollback()
		return err
	}
	return nil
}

// ExecPrepareV2 批量执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用官方的语句参数分离写法，用于批量执行相同语句
//
// s: sql语句
//
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecPrepareV2(s string, paramNum int, params ...interface{}) (int64, []int64, error) {
	p.execLocker.Lock()
	defer func() {
		if err := recover(); err != nil {
			p.Logger.Error("ExecPrepareV2 Err: " + err.(error).Error())
		}
		p.execLocker.Unlock()
	}()
	if paramNum == 0 {
		paramNum = strings.Count(s, "?")
	}

	l := len(params)
	if l%paramNum != 0 {
		return 0, nil, errors.New("not enough params")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	defer cancel()
	// 开启事务
	// st, err := p.connPool.PrepareContext(ctx, s)
	// if err != nil {
	// 	return 0, nil, err
	// }
	// defer st.Close()
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			p.Logger.Error(err.Error())
		}
	}()
	rowAffected := int64(0)
	// var ex error
	insertID := make([]int64, len(params)/paramNum)
	idx := 0
	for i := 0; i < l; i += paramNum {
		ans, err := tx.ExecContext(ctx, s, params[i:i+paramNum]...)
		// ans, err := tx.StmtContext(ctx, st).Exec(params[i : i+paramNum]...)
		if err != nil {
			// ex = err
			// continue
			return 0, nil, err
			// return rowAffected, insertID, err
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
		// tx.Rollback()
		return 0, nil, err
	}
	return rowAffected, insertID, nil
}

// ExecBatch (maybe unsafe)事务执行语句（insert，delete，update）
//
// s: sql语句
//
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
//
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecBatch(s []string) (err error) {
	p.execLocker.Lock()
	defer func() error {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return err
		}
		p.execLocker.Unlock()
		return nil
	}()
	// 检查语句，有任意语句存在风险，全部语句均不执行
	for _, v := range s {
		if err := p.checkSQL(v); err != nil {
			return err
		}
	}
	// 开启事务
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(p.Timeout))
	defer cancel()
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			p.Logger.Error(err.Error())
		}
	}()
	for _, v := range s {
		_, err = tx.ExecContext(ctx, v)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		// tx.Rollback()
		return err
	}
	return nil
}
