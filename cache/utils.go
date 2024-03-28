/*
Package cache : 数据缓存模块，可定时清理过期数据

Usage

	package main

	import (
		"github.com/xyzj/gopsu/cache"
	)
	type aaa struct{
		Name string
	}
	func main() {
		mycache:=cache.NewEmptyCache[aaa](time.Second*60) //  max 1000 data
		defer mycache.Close() // clean up and stop zhe data expire check
		mycache.Store("123","abc")
		v,ok:=mycache.Load("123")
		if !ok{
			println("key not found")
			return
		}
		println(v)
	}
*/
package cache

import (
	"time"
)

type Cache[T any] interface {
	Close()
	Clean()
	Len() int
	Extension(key string)
	Store(key string, value T) error
	StoreWithExpire(key string, value T, expire time.Duration) error
	Load(key string) (T, bool)
	LoadOrStore(key string, value T) (T, bool)
	Delete(key string)
	ForEach(f func(key string, value T) bool)
}

func NewEmptyCache() *EmptyCache[struct{}] {
	return &EmptyCache[struct{}]{}
}

// EmptyCache 一个空的cache，不实现任何功能
type EmptyCache[T any] struct{}

// SetCleanUp 设置清理周期，不低于1秒
func (ac *EmptyCache[T]) SetCleanUp(cleanup time.Duration) {}

// Close 关闭这个缓存，如果需要再次使用，应调用NewEmptyCache方法重新初始化
func (ac *EmptyCache[T]) Close() {}

// Clean 清空这个缓存
func (ac *EmptyCache[T]) Clean() {}

// Len 返回缓存内容数量
func (ac *EmptyCache[T]) Len() int {
	return 0
}

// Extension 将指定缓存延期
func (ac *EmptyCache[T]) Extension(key string) {}

// Store 添加缓存内容，如果缓存已关闭，会返回错误
func (ac *EmptyCache[T]) Store(key string, value T) error {
	return nil
}

// StoreWithExpire 添加缓存内容，设置自定义的有效时间，如果缓存已关闭，会返回错误
func (ac *EmptyCache[T]) StoreWithExpire(key string, value T, expire time.Duration) error {
	return nil
}

// Load 读取一个缓存内容，如果不存在，返回false
func (ac *EmptyCache[T]) Load(key string) (T, bool) {
	x := new(T)
	return *x, false
}

// LoadOrStore 读取或者设置一个缓存内如
//
//	当key存在时，返回缓存内容，并设置true
//	当key不存在时，将内容加入缓存，返回设置内容，并设置false
func (ac *EmptyCache[T]) LoadOrStore(key string, value T) (T, bool) {
	x := new(T)
	return *x, false
}

// Delete 删除一个缓存内容
func (ac *EmptyCache[T]) Delete(key string) {}

// ForEach 遍历所有缓存内容
func (ac *EmptyCache[T]) ForEach(f func(key string, value T) bool) {}
