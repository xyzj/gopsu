package cache

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
	"github.com/xyzj/gopsu/mapfx"
)

type cData[T any] struct {
	expire time.Time
	data   *T
}

// AnyCache 泛型结构缓存
type AnyCache[T any] struct {
	cache        *mapfx.StructMap[string, cData[T]]
	cacheCleanup *time.Ticker
	closed       atomic.Bool
	cacheExpire  time.Duration
}

// NewAnyCache 初始化一个新的缓存
//
//	 这个新缓存会创建一个线程检查内容是否过期，因此，当不再使用该缓存时，应该调用Close()方法关闭缓存
//		默认每分钟清理一次过期缓存
func NewAnyCache[VALUE any](expire time.Duration) *AnyCache[VALUE] {
	x := &AnyCache[VALUE]{
		cacheExpire:  expire,
		cache:        mapfx.NewStructMap[string, cData[VALUE]](),
		closed:       atomic.Bool{},
		cacheCleanup: time.NewTicker(time.Minute),
	}
	go loopfunc.LoopFunc(func(params ...interface{}) {
		for !x.closed.Load() {
			<-x.cacheCleanup.C
			tnow := time.Now()
			keys := make([]string, 0, x.cache.Len())
			for k, v := range x.cache.Clone() {
				if tnow.After(v.expire) {
					keys = append(keys, k)
				}
			}
			x.cache.DeleteMore(keys...)
		}
	}, "any cache", logger.NewConsoleWriter())
	return x
}

// SetCleanUp 设置清理周期，不低于1秒
func (ac *AnyCache[T]) SetCleanUp(cleanup time.Duration) {
	if cleanup < time.Second {
		cleanup = time.Second
	}
	ac.cacheCleanup.Reset(cleanup)
}

// Close 关闭这个缓存，如果需要再次使用，应调用NewAnyCache方法重新初始化
func (ac *AnyCache[T]) Close() {
	ac.closed.Store(true)
	ac.cacheCleanup.Stop()
	ac.cache.Clean()
	ac.cache = nil
}

// Clean 清空这个缓存
func (ac *AnyCache[T]) Clean() {
	if ac.closed.Load() {
		return
	}
	ac.cache.Clean()
}

// Len 返回缓存内容数量
func (ac *AnyCache[T]) Len() int {
	if ac.closed.Load() {
		return 0
	}
	return ac.cache.Len()
}

// Extension 将指定缓存延期
func (ac *AnyCache[T]) Extension(key string) {
	if x, ok := ac.cache.LoadForUpdate(key); ok {
		x.expire = time.Now().Add(ac.cacheExpire)
	}
}

// Store 添加缓存内容，如果缓存已关闭，会返回错误
func (ac *AnyCache[T]) Store(key string, value *T) error {
	return ac.StoreWithExpire(key, value, ac.cacheExpire)
}

// StoreWithExpire 添加缓存内容，设置自定义的有效时间，如果缓存已关闭，会返回错误
func (ac *AnyCache[T]) StoreWithExpire(key string, value *T, expire time.Duration) error {
	if ac.closed.Load() {
		return fmt.Errorf("cache is closed")
	}
	ac.cache.Store(key, &cData[T]{
		expire: time.Now().Add(expire),
		data:   value,
	})
	return nil
}

// Load 读取一个缓存内容，如果不存在，返回false
func (ac *AnyCache[T]) Load(key string) (*T, bool) {
	if ac.closed.Load() {
		return nil, false
	}
	v, ok := ac.cache.Load(key)
	if !ok {
		return nil, false
	}
	if time.Now().After(v.expire) {
		// ac.cache.Delete(key) // 删除会有锁操作，因此还是放在清理方法里一次性做
		return nil, false
	}
	return v.data, true
}

// LoadOrStore 读取或者设置一个缓存内如
//
//	当key存在时，返回缓存内容，并设置true
//	当key不存在时，将内容加入缓存，返回设置内容，并设置false
func (ac *AnyCache[T]) LoadOrStore(key string, value *T) (*T, bool) {
	if ac.closed.Load() {
		return nil, false
	}
	v, ok := ac.Load(key)
	if !ok {
		ac.cache.Store(key, &cData[T]{
			expire: time.Now().Add(ac.cacheExpire),
			data:   value,
		})
		return value, false
	}
	return v, true
}

// Delete 删除一个缓存内容
func (ac *AnyCache[T]) Delete(key string) {
	if ac.closed.Load() {
		return
	}
	ac.cache.Delete(key)
}

// ForEach 遍历所有缓存内容
func (ac *AnyCache[T]) ForEach(f func(key string, value *T) bool) {
	ac.cache.ForEach(func(key string, value *cData[T]) bool {
		if time.Now().After(value.expire) {
			return true
		}
		return f(key, value.data)
	})
}
