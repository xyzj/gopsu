package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ExecBatch (maybe unsafe)事务执行多个语句（insert，delete，update）
//
// s: sql语句,不支持占位符，需要使用完整语句
func (d *Conn) ExecBatch(s []string) (err error) {
	sqldb, err := d.SQLDB(d.defaultDB)
	if err != nil {
		return err
	}
	defer func() error {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return err
		}
		return nil
	}()
	// 检查语句，有任意语句存在风险，全部语句均不执行
	for _, v := range s {
		if err := checkSQL(v); err != nil {
			return err
		}
	}
	// 开启事务
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer d.rollbackCheck(tx)
	for _, v := range s {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, v)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// Exec 执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用事务
//
// s: sql语句
// params: 查询参数,对应查询语句中的`？`占位符
func (d *Conn) Exec(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	return d.ExecByDB(d.defaultDB, s, params...)
}

// ExecByDB 执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用事务
//
// dbidx: 指定数据库名称
// s: sql语句
// params: 查询参数,对应查询语句中的`？`占位符
func (d *Conn) ExecByDB(dbidx int, s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	sqldb, err := d.SQLDB(dbidx)
	if err != nil {
		return 0, 0, err
	}
	defer func() (int64, int64, error) {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return 0, 0, err
		}
		return rowAffected, insertID, nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	defer cancel()
	// res, err := sqldb.ExecContext(ctx, s, params...)
	// if err != nil {
	// 	return 0, 0, err
	// }
	// 开启事务
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer d.rollbackCheck(tx)
	res, err := tx.ExecContext(ctx, s, params...)
	if err != nil {
		return 0, 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, 0, err
	}
	insertID, _ = res.LastInsertId()
	rowAffected, _ = res.RowsAffected()
	return rowAffected, insertID, nil
}

// ExecPrepare 批量执行占位符语句,用于批量执行语句相同但数据内容不同的场景
//
// s: sql语句
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
// params: 查询参数,对应查询语句中的`？`占位符
func (d *Conn) ExecPrepare(s string, paramNum int, params ...interface{}) (err error) {
	return d.ExecPrepareByDB(d.defaultDB, s, paramNum, params...)
}

// ExecPrepareByDB 批量执行占位符语句,用于批量执行语句相同但数据内容不同的场景
//
// dbidx: 数据库名称
// s: sql语句
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
// params: 查询参数,对应查询语句中的`？`占位符
func (d *Conn) ExecPrepareByDB(dbidx int, s string, paramNum int, params ...interface{}) (err error) {
	sqldb, err := d.SQLDB(dbidx)
	if err != nil {
		return err
	}
	defer func() error {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return err
		}
		return nil
	}()
	if paramNum == 0 {
		paramNum = strings.Count(s, "?")
	}

	l := len(params)
	if l%paramNum != 0 {
		return errors.New("not enough params")
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	defer cancel()
	// 开启事务
	tx, err := sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer d.rollbackCheck(tx)
	for i := 0; i < l; i += paramNum {
		_, err := tx.ExecContext(ctx, s, params[i:i+paramNum]...)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (d *Conn) rollbackCheck(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		d.cfg.Logger.Error("[DB] " + err.Error())
	}
}
