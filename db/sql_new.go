/*
Package db : 数据库模块，封装了常用方法，可缓存数据，可依据配置自动创建myisam引擎的子表，支持mysql和sqlserver
*/
package db

// for greatsql ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY 'root';
import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	mydsn "github.com/go-sql-driver/mysql"
	"github.com/microsoft/go-mssqldb/msdsn"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/cache"
	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/json"
	"github.com/xyzj/gopsu/logger"
	"gorm.io/driver/mysql"
	mssql "gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

var codeGzip = gopsu.GetNewArchiveWorker(gopsu.ArchiveGZip)

// SQLInterface 数据库接口
type SQLInterface interface {
	IsReady() bool
	QueryCacheJSON(string, int, int) string
	QueryCachePB2(string, int, int) *QueryData
	Exec(string, ...interface{}) (int64, int64, error)
	ExecPrepare(string, int, ...interface{}) error
}

type Drive string

const (
	DriveMySQL     Drive = "mysql"
	DriveSQLServer Drive = "sqlserver"
	DrivePostgre   Drive = "postgre"

	emptyCacheTag = "00000-0"
)

type Opt struct {
	// 数据驱动
	DriverType Drive
	// 服务地址
	Server string
	// 用户名
	User string
	// 密码
	Passwd string
	// tls 参数
	TLS string
	// 数据库名称
	DBNames []string
	// 数据库初始化脚本，和DBName对应
	InitScripts []string
	// 设置缓存
	QueryCache cache.Cache[*QueryData]
	// 日志
	Logger logger.Logger
	// 执行超时
	Timeout     time.Duration
	enableCache bool
}

// QueryDataChan chan方式返回首页数据
type QueryDataChan struct {
	Locker *sync.WaitGroup
	Data   *QueryData
	Total  *int
	Err    error
}

func newDataRow(cols int) *QueryDataRow {
	return &QueryDataRow{
		Cells:  make([]string, cols),
		VCells: make([]*config.Value, cols),
		// vCell:  make([]*VCell, cols),
	}
}

// QueryDataRow 数据行
type QueryDataRow struct {
	// Deprecated: will removed in a future version, use VCells
	Cells  []string        `json:"cells,omitempty"`
	VCells []*config.Value `json:"vcells,omitempty"`
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

type dbs struct {
	ormdb  *gorm.DB
	sqldb  *sql.DB
	name   string
	dbtype string
}

// Conn sql连接池
type Conn struct {
	cfg *Opt
	dbs map[int]*dbs
	// 缓存路径
	cacheDir  string
	cacheHead string
	defaultDB int
	isnew     bool
}

// New 新的sql连接池
func New(opt *Opt) (*Conn, error) {
	if opt == nil {
		return nil, fmt.Errorf("config error")
	}
	if opt.Server == "" || opt.User == "" || len(opt.DBNames) == 0 {
		return nil, fmt.Errorf("config error")
	}
	if opt.Logger == nil {
		opt.Logger = &logger.NilLogger{}
	}
	if opt.Timeout == 0 {
		opt.Timeout = time.Second * 300
	}
	if opt.QueryCache == nil {
		opt.enableCache = false
		opt.QueryCache = &cache.EmptyCache[*QueryData]{}
	} else {
		opt.enableCache = true
	}
	l1 := len(opt.DBNames)
	l2 := len(opt.InitScripts)
	for i := l2; i < l1; i++ {
		opt.InitScripts = append(opt.InitScripts, "")
	}
	d := &Conn{
		dbs:       make(map[int]*dbs),
		cfg:       opt,
		defaultDB: 1,
	}
	var connstr string
	var orm *gorm.DB
	var err error
	reConn := 0
CONN:
	dbidx := 1
	var name, value, dbtype string
	for k, dbname := range opt.DBNames {
		dbname = strings.TrimSpace(dbname)
		if dbname == "" {
			continue
		}
		switch opt.DriverType {
		case DriveSQLServer:
			ss := strings.Split(opt.Server, ":")
			if len(ss) == 1 {
				ss = append(ss, "1433")
			}
			pp, err := strconv.ParseUint(ss[1], 10, 64)
			if err != nil {
				pp = 1433
			}
			connstr = msdsn.Config{
				Host:        ss[0],
				Port:        pp,
				User:        opt.User,
				Password:    opt.Passwd,
				Database:    dbname,
				DialTimeout: time.Second * 10,
				ConnTimeout: time.Second * 10,
			}.URL().String()
			orm, err = gorm.Open(mssql.Open(connstr))
			if err != nil {
				return nil, err
			}
		case DriveMySQL:
			sqlcfg := &mydsn.Config{
				Collation:            "utf8_general_ci",
				Loc:                  time.Local,
				MaxAllowedPacket:     0, // 64*1024*1024
				AllowNativePasswords: true,
				CheckConnLiveness:    true,
				Net:                  "tcp",
				Addr:                 opt.Server,
				User:                 opt.User,
				Passwd:               opt.Passwd,
				DBName:               dbname,
				MultiStatements:      true,
				ParseTime:            true,
				Timeout:              time.Second * 180,
				ClientFoundRows:      true,
				InterpolateParams:    true,
				TLSConfig:            opt.TLS,
			}
			connstr = sqlcfg.FormatDSN()
			orm, err = gorm.Open(mysql.Open(connstr))
			if err != nil {
				if !strings.Contains(err.Error(), "Unknown database") || reConn > 0 {
					return nil, err
				}
				sqlcfg.DBName = "mysql"
				dd, err := sql.Open(string(opt.DriverType), strings.ReplaceAll(sqlcfg.FormatDSN(), "\n", ""))
				if err != nil {
					return nil, err
				}
				defer dd.Close()
				_, err = dd.Exec("CREATE DATABASE IF NOT EXISTS `" + dbname + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;use `" + dbname + "`;")
				if err != nil {
					return nil, err
				}
				opt.Logger.System("[DB] Create database `" + dbname + "` on " + opt.Server)
				if opt.InitScripts[k] != "" {
					_, err = dd.Exec(opt.InitScripts[k])
					if err != nil {
						return nil, err
					}
					opt.Logger.System("[DB] Create tables in " + opt.Server + "/" + dbname)
				}
				d.isnew = true
				reConn++
				goto CONN
			}
		default:
			return nil, fmt.Errorf("not support yet")
		}
		reConn = 0
		sqldb, err := orm.DB()
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		if err = sqldb.PingContext(ctx); err != nil {
			return nil, err
		}
		if dbtype == "" {
			err = sqldb.QueryRow("show variables like 'version_comment';").Scan(&name, &value)
			if err != nil {
				dbtype = "unknow"
			} else {
				switch {
				case strings.Contains(strings.ToLower(value), "mariadb"):
					dbtype = "mariadb"
				case strings.Contains(strings.ToLower(value), "mysql"):
					dbtype = "mysql"
				case strings.Contains(strings.ToLower(value), "greatsql"):
					dbtype = "greatsql"
				}
			}
		}
		d.dbs[dbidx] = &dbs{
			name:   dbname,
			ormdb:  orm,
			sqldb:  sqldb,
			dbtype: dbtype,
		}
		dbidx++
		d.cacheHead = gopsu.CalcCRC32String([]byte(connstr))
		d.cacheDir = gopsu.DefaultCacheDir
	}
	d.cfg.Logger.System("[DB] Success connect to server " + d.cfg.Server)
	return d, nil
}

func (d *Conn) TablesAreNew() bool {
	return d.isnew
}

// GetDBIdx 连接多个数据库的时候，设置默认的数据库名称
func (d *Conn) GetDBIdx(dbname string) (*sql.DB, error) {
	for _, v := range d.dbs {
		if v.name == dbname {
			return v.sqldb, nil
		}
	}
	return nil, fmt.Errorf(dbname + " not found")
}

// SetDefaultDB 连接多个数据库的时候，设置默认的数据库名称
func (d *Conn) SetDefaultDB(dbidx int) error {
	if v, ok := d.dbs[dbidx]; !ok {
		return fmt.Errorf(v.name + " not found")
	}
	d.defaultDB = dbidx
	return nil
}

func (d *Conn) DBType() string {
	if len(d.dbs) == 0 {
		return "unknow"
	}
	return d.dbs[1].dbtype
}

func (d *Conn) MaxDBIdx() int {
	return len(d.dbs)
}

// ORM 指定要返回的gorm.db实例
//
// dbname: 数据库名称，不设置时返回默认
func (d *Conn) ORM(dbidx int) (*gorm.DB, error) {
	v, ok := d.dbs[dbidx]
	if ok {
		return v.ormdb, nil
	}
	return nil, fmt.Errorf(v.name + " not found")
}

// SQLDB 指定要返回的sql.db实例
//
// dbname: 数据库名称，不设置时返回默认
func (d *Conn) SQLDB(dbidx int) (*sql.DB, error) {
	v, ok := d.dbs[dbidx]
	if ok {
		return v.sqldb, nil
	}
	return nil, fmt.Errorf(v.name + " not found")
}

// IsReady 检查状态，仅检查默认库的状态
func (d *Conn) IsReady() bool {
	if len(d.dbs) == 0 {
		return false
	}
	sql, ok := d.dbs[d.defaultDB]
	if !ok {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := sql.sqldb.PingContext(ctx); err != nil {
		return false
	}
	return true
}

// GetName 获取数据库名字
func (d *Conn) GetName(dbidx int) string {
	if dbidx > len(d.dbs) {
		return ""
	}
	sql, ok := d.dbs[dbidx]
	if !ok {
		return ""
	}
	return sql.name
}

// checkSQL 检查sql语句是否存在注入攻击风险
//
// args：
//
// s： sql语句
func checkSQL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if gopsu.CheckSQLInject(s) {
		return nil
	}
	return fmt.Errorf("SQL statement has risk of injection: " + s)
}

func newResult() *QueryData {
	return &QueryData{
		Columns: []string{},
		Rows:    []*QueryDataRow{},
	}
}

const duplicateKey = "ON DUPLICATE KEY UPDATE"

var dupReplacer = strings.NewReplacer(")", "",
	"=values(", "=autoalias.",
	"=VALUES(", "=autoalias.",
	"=Values(", "=autoalias.")

// MariadbDuplicate2Mysql 将mariadb的insert on duplicate语句修改为mysql的样式
// 注意：`ON DUPLICATE KEY UPDATE` 需要全大写，用于识别
func (d *Conn) MariadbDuplicate2Mysql(strsql string) string {
	if d.DBType() == "mariadb" {
		return strsql
	}
	if !strings.Contains(strsql, duplicateKey) { // 不包含就返回原始语句
		return strsql
	}
	ss := strings.Split(strsql, duplicateKey)
	if strings.Contains(strings.ToLower(ss[1]), "=values(") { // 包含mariadb语句特征
		return ss[0] + " as autoalias " + duplicateKey + " " + dupReplacer.Replace(ss[1])
	}
	return strsql
}
