package db

import (
	"time"

	"github.com/xyzj/gopsu/json"
	"go.etcd.io/bbolt"
)

// BoltDB bolt数据文件实例
type BoltDB struct {
	db       *bbolt.DB
	bucket   []byte
	filename string
}

// Close 关闭文件数据库
func (b *BoltDB) Close() error {
	return b.db.Close()
}

// Read 读取一个值
func (b *BoltDB) Read(key string, bucket ...string) string {
	var buc []byte
	if len(bucket) == 0 {
		buc = b.bucket
	} else {
		if bucket[0] == "" {
			buc = b.bucket
		} else {
			buc = json.Bytes(bucket[0])
		}
	}
	var value string
	b.db.View(func(tx *bbolt.Tx) error {
		t := tx.Bucket(buc)
		if t == nil {
			value = ""
			return nil
		}
		b := t.Get(json.Bytes(key))
		if b == nil {
			value = ""
		} else {
			value = json.String(b)
		}
		return nil
	})
	return value
}

// Write 写入一个值
func (b *BoltDB) Write(key, value string, bucket ...string) error {
	var buc []byte
	if len(bucket) == 0 {
		buc = b.bucket
	} else {
		if bucket[0] == "" {
			buc = b.bucket
		} else {
			buc = json.Bytes(bucket[0])
		}
	}
	return b.db.Update(func(tx *bbolt.Tx) error {
		t, err := tx.CreateBucketIfNotExists(buc)
		if err != nil {
			return err
		}
		return t.Put(json.Bytes(key), json.Bytes(value))
	})
}

// Delete 删除一个值
func (b *BoltDB) Delete(key string, bucket ...string) error {
	var buc []byte
	if len(bucket) == 0 {
		buc = b.bucket
	} else {
		if bucket[0] == "" {
			buc = b.bucket
		} else {
			buc = json.Bytes(bucket[0])
		}
	}
	return b.db.Update(func(tx *bbolt.Tx) error {
		t := tx.Bucket(buc)
		if t == nil {
			return nil
		}
		return t.Delete(json.Bytes(key))
	})
}

// ForEach 遍历所有key,value
func (b *BoltDB) ForEach(f func(k, v string) error, bucket ...string) {
	var buc []byte
	if len(bucket) == 0 {
		buc = b.bucket
	} else {
		if bucket[0] == "" {
			buc = b.bucket
		} else {
			buc = json.Bytes(bucket[0])
		}
	}
	var data = make(map[string]string)
	b.db.View(func(tx *bbolt.Tx) error {
		t := tx.Bucket(buc)
		if t == nil {
			return nil
		}
		return t.ForEach(func(k, v []byte) error {
			data[json.String(k)] = json.String(v)
			return nil
			// defer func() {
			// 	recover()
			// }()
			// return f(json.String(k), json.String(v))
		})
	})
	defer func() {
		recover()
	}()
	for k, v := range data {
		f(k, v)
	}
}

// NewBolt 创建一个新的bolt数据文件
func NewBolt(f string) (*BoltDB, error) {
	db, err := bbolt.Open(f, 0664, &bbolt.Options{Timeout: time.Second * 2})
	if err != nil {
		return nil, err
	}

	return &BoltDB{
		db:       db,
		bucket:   json.Bytes("default"),
		filename: f,
	}, nil
}
