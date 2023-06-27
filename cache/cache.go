/*
Package cache : 数据缓存模块，可定时清理过期数据

Usage

	package main

	import (
		"github.com/xyzj/gopsu/cache"
	)

	func main() {
		mycache:=cache.NewCache(1000) //  max 1000 data
		defer mycache.End() // clean up and stop zhe data expire check
		mycache.Set("123","abc")
		v,ok:=mycache.Get("123")
		if !ok{
			println("key not found")
			return
		}
		println(v)
	}
*/
package cache

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/xyzj/gopsu/loopfunc"
)

type mapCache struct {
	locker sync.RWMutex
	data   map[string]*xCacheData
}

func newMap() *mapCache {
	return &mapCache{
		locker: sync.RWMutex{},
		data:   make(map[string]*xCacheData),
	}
}
func (m *mapCache) store(key string, value interface{}, expire time.Duration) {
	m.locker.Lock()
	m.data[key] = &xCacheData{
		Expire: time.Now().Add(expire),
		Value:  value,
	}
	m.locker.Unlock()
}
func (m *mapCache) expire(key string, expire time.Duration) bool {
	m.locker.Lock()
	defer m.locker.Unlock()
	v, ok := m.data[key]
	if ok {
		v.Expire = time.Now().Add(expire)
		return true
		// m.data[key] = v
	}
	return false
}
func (m *mapCache) load(key string) (interface{}, bool) {
	m.locker.RLock()
	x, ok := m.data[key]
	m.locker.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(x.Expire) {
		return nil, false
	}
	return x.Value, true
}
func (m *mapCache) del(key string) {
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}
func (m *mapCache) len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}
func (m *mapCache) clean() {
	m.locker.Lock()
	m.data = make(map[string]*xCacheData)
	m.locker.Unlock()
}
func (m *mapCache) xrange(f func(key string, value interface{}, expire time.Time) bool) {
	m.locker.RLock()
	x := deepcopy.Copy(m.data).(map[string]*xCacheData)
	m.locker.RUnlock()
	defer func() {
		if err := recover(); err != nil {
			println("cache range error: " + err.(error).Error())
		}
	}()
	for k, v := range x {
		if !f(k, v.Value, v.Expire) {
			break
		}
	}
}

// xCacheData 可设置超时的缓存字典数据结构
type xCacheData struct {
	Value  interface{}
	Expire time.Time
}

// XCache 可设置超时的缓存字典
type XCache struct {
	max  int
	data *mapCache
	end  chan string
}

// End 关掉这个cache
func (xc *XCache) End() {
	xc.end <- "end"
}

// NewCacheWithWriter 创建新的缓存字典,并指定panic日志输出writer
//
// max：字典大小,0-不限制(谨慎使用)
func NewCacheWithWriter(max int, logw io.Writer) *XCache {
	xc := &XCache{
		max:  max,
		data: newMap(),
		end:  make(chan string, 1),
	}
	if logw == nil {
		logw = os.Stdout
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		t := time.NewTicker(time.Minute)
		for {
			select {
			case <-t.C:
				xc.data.xrange(func(key string, value interface{}, expire time.Time) bool {
					if time.Now().After(expire) {
						xc.data.del(key)
					}
					return true
				})
			case <-xc.end:
				xc.data.clean()
				return
			}
		}
	}, "mem cache expire", logw)
	return xc
}

// NewCache 创建新的缓存字典
//
// max：字典大小,0-不限制(谨慎使用)
func NewCache(max int) *XCache {
	return NewCacheWithWriter(max, os.Stdout)
}

// Set 设置缓存数据
//
// k: key
//
// v: value
//
// expire: 超时时间，有效单位秒
func (xc *XCache) Set(k string, v interface{}, expire time.Duration) bool {
	if xc.max > 0 && xc.data.len() >= xc.max {
		return false
	}
	xc.data.store(k, v, expire)
	return true
}

// SetWithHold 设置缓存数据
//
// k: key
//
// v: value
//
// expire: 超时时间，有效单位秒
//
// timeout: 写入超时，有效单位秒
func (xc *XCache) SetWithHold(k string, v interface{}, expire, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	t := time.NewTicker(time.Millisecond * 300)
	for {
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
			if xc.Set(k, v, expire) {
				return true
			}
		}
	}
}

// Get 读取缓存数据
func (xc *XCache) Get(k string) (interface{}, bool) {
	return xc.data.load(k)
}

// GetAndExpire 读取缓存数据，并延长缓存时效
func (xc *XCache) GetAndExpire(k string, expire time.Duration) (interface{}, bool) {
	v, ok := xc.data.load(k)
	if ok {
		xc.data.expire(k, expire)
		return v, ok
	}
	return nil, false
}

// GetAndRemove 读取缓存数据，并删除
func (xc *XCache) GetAndRemove(k string) (interface{}, bool) {
	v, ok := xc.data.load(k)
	if ok {
		xc.data.del(k)
		return v, ok
	}
	return nil, false
}

// Clean 清空缓存
func (xc *XCache) Clean() {
	xc.data.clean()
}

// Len 获取缓存数量
func (xc *XCache) Len() int {
	return xc.data.len()
}
