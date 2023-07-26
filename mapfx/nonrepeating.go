package mapfx

import (
	"reflect"
	"sync"
)

// 使用示例
//
// a := NewUniqueSlice[string]()
// for _, v := range []string{"1", "2", "3", "3", "4", "5"} {
// 	a.Store(v)
// }
// println(a.Slice()) // ["1", "2", "3", "4", "5"]
//
// for _, v := range []string{"1", "2", "3", "3", "4", "5"} {
// 	if a.Store(v) { // 判断一下如果成功插入，说明原来没有值，可以进行下一步操作之类的
// 		// do something
// 	}
// }

// NewUniqueSlice 利用map构建一个内容不重复的slice
//
//	可设置是否线程安全
func NewUniqueSlice[T byte | int8 | int | int32 | int64 | float32 | float64 | string](goroutineSafe bool) *UniqueSlice[T] {
	return &UniqueSlice[T]{
		locker: sync.RWMutex{},
		data:   make(map[T]struct{}),
		safe:   goroutineSafe,
	}
}

// UniqueSlice 一个不重复的切片结构
type UniqueSlice[T byte | int8 | int | int32 | int64 | float32 | float64 | string] struct {
	locker sync.RWMutex
	data   map[T]struct{}
	safe   bool
}

func (u *UniqueSlice[T]) Store(item T) bool {
	if u.safe {
		u.locker.Lock()
		defer u.locker.Unlock()
	}
	if _, ok := u.data[item]; ok {
		return false
	}
	u.data[item] = struct{}{}
	return true
}
func (u *UniqueSlice[T]) StoreMany(items ...T) {
	if u.safe {
		u.locker.Lock()
		defer u.locker.Unlock()
	}
	for _, item := range items {
		if _, ok := u.data[item]; ok {
			continue
		}
		u.data[item] = struct{}{}
	}
}
func (u *UniqueSlice[T]) Clean() {
	if u.safe {
		u.locker.Lock()
		defer u.locker.Unlock()
	}
	u.data = make(map[T]struct{})
}
func (u *UniqueSlice[T]) Len() int {
	if u.safe {
		u.locker.RLock()
		defer u.locker.RUnlock()
	}
	return len(u.data)
}
func (u *UniqueSlice[T]) Slice() []T {
	if u.safe {
		u.locker.RLock()
		defer u.locker.RUnlock()
	}
	x := make([]T, 0, len(u.data))
	for k := range u.data {
		x = append(x, k)
	}
	return x
}
func (u *UniqueSlice[T]) Has(item T) bool {
	if u.safe {
		u.locker.RLock()
		defer u.locker.RUnlock()
	}
	for k := range u.data {
		if k == item {
			return true
		}
	}
	return false
}

// NewUniqueStructSlice 构建一个内容不重复的struct 类型 slice
//
//	线程安全
func NewUniqueStructSlice[T StructMapI](goroutineSafe bool) *UniqueStructSlice[T] {
	return &UniqueStructSlice[T]{
		safe:   goroutineSafe,
		locker: sync.RWMutex{},
		data:   make([]T, 0),
	}
}

// UniqueStructSlice 一个不重复的struct切片结构
type UniqueStructSlice[T StructMapI] struct {
	locker sync.RWMutex
	data   []T
	safe   bool
}

func (u *UniqueStructSlice[T]) Store(item T) bool {
	if u.safe {
		u.locker.Lock()
		defer u.locker.Unlock()
	}
	for _, v := range u.data {
		if reflect.DeepEqual(v, item) {
			return false
		}
	}
	u.data = append(u.data, item)
	return true
}
func (u *UniqueStructSlice[T]) StoreMany(items ...T) {
	if u.safe {
		u.locker.Lock()
		defer u.locker.Unlock()
	}
	for _, item := range items {
		found := false
		for _, v := range u.data {
			if reflect.DeepEqual(v, item) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		u.data = append(u.data, item)
	}
}
func (u *UniqueStructSlice[T]) Clean() {
	if u.safe {
		u.locker.Lock()
		defer u.locker.Unlock()
	}
	u.data = make([]T, 0)
}
func (u *UniqueStructSlice[T]) Len() int {
	if u.safe {
		u.locker.RLock()
		defer u.locker.RUnlock()
	}
	return len(u.data)
}
func (u *UniqueStructSlice[T]) Slice() []T {
	if u.safe {
		u.locker.RLock()
		defer u.locker.RUnlock()
	}
	x := make([]T, 0, len(u.data))
	x = append(x, u.data...)
	return x
}
func (u *UniqueStructSlice[T]) Has(item T) bool {
	if u.safe {
		u.locker.RLock()
		defer u.locker.RUnlock()
	}
	for _, k := range u.data {
		if reflect.DeepEqual(k, item) {
			return true
		}
	}
	return false
}
