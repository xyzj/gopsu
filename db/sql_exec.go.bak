package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Exec 执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用官方的语句参数分离写法
//
// s: sql语句
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) Exec(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	defer func() (int64, int64, error) {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return 0, 0, err
		}
		return rowAffected, insertID, nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
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
// Deprecated: use ExecTx()
func (p *SQLPool) ExecV2(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	return p.ExecTx(s, params...)
}

// ExecTx 事务执行语句（insert，delete，update），可回滚,返回（影响行数,insertId,error）,使用官方的语句参数分离写法
//
// s: sql语句
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecTx(s string, params ...interface{}) (rowAffected, insertID int64, err error) {
	defer func() (int64, int64, error) {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return 0, 0, err
		}
		return rowAffected, insertID, nil
	}()
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()
	// 开启事务
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer p.rollbackCheck(tx)
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

// ExecPrepare 批量执行占位符语句 返回 err，使用官方的语句参数分离写法，用于批量执行相同语句
//
// s: sql语句
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecPrepare(s string, paramNum int, params ...interface{}) (err error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()
	// 开启事务
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && err != sql.ErrTxDone {
			p.Logger.Error("[DB] " + err.Error())
		}
	}()
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

// ExecPrepareV2 批量执行语句（insert，delete，update）,返回（影响行数,insertId,error）,使用官方的语句参数分离写法，用于批量执行相同语句
//
// s: sql语句
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecPrepareV2(s string, paramNum int, params ...interface{}) (int64, []int64, error) {
	defer func() {
		if err := recover(); err != nil {
			p.Logger.Error("[DB] ExecPrepareV2 Err: " + err.(error).Error())
		}
	}()
	if paramNum == 0 {
		paramNum = strings.Count(s, "?")
	}

	l := len(params)
	if l%paramNum != 0 {
		return 0, nil, errors.New("not enough params")
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()
	// 开启事务
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer p.rollbackCheck(tx)
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

// ExecBatch (maybe unsafe)事务执行语句（insert，delete，update）
//
// s: sql语句
// paramNum: 占位符数量,为0时自动计算sql语句中`?`的数量
// params: 查询参数,对应查询语句中的`？`占位符
func (p *SQLPool) ExecBatch(s []string) (err error) {
	defer func() error {
		if ex := recover(); ex != nil {
			err = ex.(error)
			return err
		}
		return nil
	}()
	// 检查语句，有任意语句存在风险，全部语句均不执行
	for _, v := range s {
		if err := p.checkSQL(v); err != nil {
			return err
		}
	}
	// 开启事务
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tx, err := p.connPool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer p.rollbackCheck(tx)
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

func (p *SQLPool) rollbackCheck(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		p.Logger.Error("[DB] " + err.Error())
	}
}
