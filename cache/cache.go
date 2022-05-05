package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// XCache 可设置超时的缓存字典
type XCache struct {
	len    int64
	am     map[string]*xCacheData
	locker *sync.RWMutex
	amIdx  int64
}

// xCacheData 可设置超时的缓存字典数据结构
type xCacheData struct {
	// key    interface{}
	value  interface{}
	expire time.Time
}

// Value 返回缓存值
func (xcd *xCacheData) Value() interface{} {
	return xcd.value
}

// NewCache 创建新的缓存字典
//	l：字典大小,0-不限制
func NewCache(l int64) *XCache {
	xc := &XCache{
		len:    l,
		locker: &sync.RWMutex{},
		am:     make(map[string]*xCacheData),
	}
	go xc.run()
	return xc
}

// Set 设置缓存数据
//	k: key
//	v: value
//	expire: 超时时间，有效单位秒
func (xc *XCache) Set(k string, v interface{}, expire time.Duration) bool {
	xc.locker.Lock()
	defer xc.locker.Unlock()
	if xc.len > 0 && xc.amIdx >= xc.len {
		return false
	}
	xc.am[k] = &xCacheData{
		// key:    k,
		value:  v,
		expire: time.Now().Add(expire),
	}
	atomic.AddInt64(&xc.amIdx, 1)
	return true
}

// SetWithHold 设置缓存数据
//	k: key
//	v: value
//	expire: 超时时间，有效单位秒
//	timeout: 写入超时，有效单位秒
func (xc *XCache) SetWithHold(k string, v interface{}, expire, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Millisecond * 300):
			if xc.Set(k, v, expire) {
				return true
			}
		}
	}
}

// Get 读取缓存数据
func (xc *XCache) Get(k string) (interface{}, bool) {
	xc.locker.RLock()
	defer xc.locker.RUnlock()
	v, ok := xc.am[k]
	if ok {
		if v.expire.Before(time.Now()) {
			return nil, false
		}
		return v.value, true
	}
	return nil, false
}

// GetAndExpire 读取缓存数据，并延长缓存时效
func (xc *XCache) GetAndExpire(k string, expire time.Duration) (interface{}, bool) {
	xc.locker.Lock()
	defer xc.locker.Unlock()
	v, ok := xc.am[k]
	if ok {
		if v.expire.Before(time.Now()) {
			delete(xc.am, k)
			atomic.AddInt64(&xc.amIdx, -1)
			return nil, false
		}
		v.expire = time.Now().Add(expire)
		return v.value, true
	}
	return nil, false
}

// GetAndRemove 读取缓存数据，并删除
func (xc *XCache) GetAndRemove(k string) (interface{}, bool) {
	xc.locker.Lock()
	defer xc.locker.Unlock()
	v, ok := xc.am[k]
	var xx interface{}
	if ok {
		if v.expire.After(time.Now()) {
			xx = v.value
		}
		delete(xc.am, k)
		atomic.AddInt64(&xc.amIdx, -1)
	}
	return xx, true
}

// Range 遍历缓存
// func (xc *XCache) Range(f func(key, value interface{}) bool) {
// 	xc.am.Range(f)
// }

// Clear 清空缓存
func (xc *XCache) Clear() {
	xc.locker.Lock()
	xc.am = make(map[string]*xCacheData)
	xc.amIdx = 0
	xc.locker.Unlock()
}

// Len 获取缓存数量
func (xc *XCache) Len() int64 {
	return xc.amIdx
}

func (xc *XCache) run() {
	// var exlocker sync.WaitGroup
	var t = time.NewTicker(time.Minute)
	// var t = time.NewTicker(time.Millisecond * 100)
RUN:
	// exlocker.Add(1)
	func() {
		defer func() {
			recover()
			// exlocker.Done()
		}()
		for range t.C {
			tt := time.Now()
			xc.locker.Lock()
			for k, v := range xc.am {
				if v.expire.Before(tt) {
					delete(xc.am, k)
				}
			}
			xc.amIdx = int64(len(xc.am))
			xc.locker.Unlock()
		}
	}()
	time.Sleep(time.Second)
	// exlocker.Wait()
	goto RUN
}
