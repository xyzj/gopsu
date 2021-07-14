package gopsu

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// XCache 可设置超时的缓存字典
type XCache struct {
	len   int64
	am    sync.Map
	amIdx int64
}

// XCacheData 可设置超时的缓存字典数据结构
type XCacheData struct {
	key    interface{}
	value  interface{}
	expire int64
}

func (xcd *XCacheData) Value() interface{} {
	return xcd.value
}

// NewCache 创建新的缓存字典
//	l：字典大小,0-不限制
func NewCache(l int64) *XCache {
	xc := &XCache{
		len: l,
	}
	go xc.run()
	return xc
}

// Set 设置缓存数据
//	k: key
//	v: value
//	expire: 超时时间（ms）,0-不超时
func (xc *XCache) Set(k, v interface{}, expire int64) bool {
	if xc.len > 0 && xc.amIdx >= xc.len {
		return false
	}
	if expire <= 0 {
		expire = 316224000000
	}
	xc.am.Store(k, &XCacheData{
		key:    k,
		value:  v,
		expire: time.Now().UnixNano()/1000000 + expire,
	})
	atomic.AddInt64(&xc.amIdx, 1)
	return true
}

// SetWithHold 设置缓存数据
//	k: key
//	v: value
//	expire: 超时时间（ms）,0-不超时
//	timeout: 写入超时，ms，0-始终等待
func (xc *XCache) SetWithHold(k, v interface{}, expire, timeout int64) bool {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*time.Duration(timeout))
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Millisecond * 3):
			if xc.Set(k, v, expire) {
				return true
			}
		}
	}
}

// Get 读取缓存数据
func (xc *XCache) Get(k interface{}) (interface{}, bool) {
	v, ok := xc.am.Load(k)
	if ok {
		return v.(*XCacheData).value, true
	}
	return nil, false
}

// Range 遍历缓存
func (xc *XCache) Range(f func(key, value interface{}) bool) {
	xc.am.Range(f)
}

// Clear 清空缓存
func (xc *XCache) Clear() {
	xc.am.Range(func(key interface{}, value interface{}) bool {
		xc.am.Delete(key)
		atomic.AddInt64(&xc.amIdx, -1)
		return true
	})
}

// Len 获取缓存数量
func (xc *XCache) Len() int64 {
	return xc.amIdx
}

func (xc *XCache) run() {
	// var exlocker sync.WaitGroup
	var t = time.NewTicker(time.Millisecond * 10)
RUN:
	// exlocker.Add(1)
	func() {
		defer func() {
			recover()
			// exlocker.Done()
		}()
		for range t.C {
			tt := time.Now().UnixNano() / 1000000
			xc.am.Range(func(key interface{}, value interface{}) bool {
				if value.(*XCacheData).expire <= tt {
					xc.am.Delete(key)
					atomic.AddInt64(&xc.amIdx, -1)
				}
				return true
			})
		}
	}()
	time.Sleep(time.Second)
	// exlocker.Wait()
	goto RUN
}
