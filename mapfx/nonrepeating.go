package mapfx

import "sync"

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
//	非线程安全，请勿跨线程使用
func NewUniqueSlice[T byte | int8 | int | int32 | int64 | float32 | float64 | string]() *UniqueSlice[T] {
	return &UniqueSlice[T]{
		// locker: sync.RWMutex{},
		data: make(map[T]struct{}),
	}
}

// UniqueSlice 一个不重复的切片结构
type UniqueSlice[T byte | int8 | int | int32 | int64 | float32 | float64 | string] struct {
	// locker sync.RWMutex
	data map[T]struct{}
}

func (u *UniqueSlice[T]) Store(item T) bool {
	// u.locker.Lock()
	// defer u.locker.Unlock()
	if _, ok := u.data[item]; ok {
		return false
	}
	u.data[item] = struct{}{}
	return true
}
func (u *UniqueSlice[T]) StoreMany(items ...T) {
	// u.locker.Lock()
	// defer u.locker.Unlock()
	for _, item := range items {
		if _, ok := u.data[item]; ok {
			continue
		}
		u.data[item] = struct{}{}
	}
}
func (u *UniqueSlice[T]) Clean() {
	u.data = make(map[T]struct{})
}
func (u *UniqueSlice[T]) Len() int {
	// u.locker.RLock()
	// defer u.locker.RUnlock()
	return len(u.data)
}
func (u *UniqueSlice[T]) Slice() []T {
	// u.locker.RLock()
	// defer u.locker.RUnlock()
	x := make([]T, 0, len(u.data))
	for k := range u.data {
		x = append(x, k)
	}
	return x
}
func (u *UniqueSlice[T]) Has(item T) bool {
	for k := range u.data {
		if k == item {
			return true
		}
	}
	return false
}

// NewUniqueSliceSafe 利用map构建一个内容不重复的slice
//
//	线程安全
func NewUniqueSliceSafe[T byte | int8 | int | int32 | int64 | float32 | float64 | string]() *UniqueSliceSafe[T] {
	return &UniqueSliceSafe[T]{
		locker: sync.RWMutex{},
		data:   make(map[T]struct{}),
	}
}

// UniqueSliceSafe 一个不重复的切片结构
type UniqueSliceSafe[T byte | int8 | int | int32 | int64 | float32 | float64 | string] struct {
	locker sync.RWMutex
	data   map[T]struct{}
}

func (u *UniqueSliceSafe[T]) Store(item T) bool {
	u.locker.Lock()
	defer u.locker.Unlock()
	if _, ok := u.data[item]; ok {
		return false
	}
	u.data[item] = struct{}{}
	return true
}
func (u *UniqueSliceSafe[T]) StoreMany(items ...T) {
	u.locker.Lock()
	defer u.locker.Unlock()
	for _, item := range items {
		if _, ok := u.data[item]; ok {
			continue
		}
		u.data[item] = struct{}{}
	}
}
func (u *UniqueSliceSafe[T]) Clean() {
	u.locker.Lock()
	defer u.locker.Unlock()
	u.data = make(map[T]struct{})
}
func (u *UniqueSliceSafe[T]) Len() int {
	u.locker.RLock()
	defer u.locker.RUnlock()
	return len(u.data)
}
func (u *UniqueSliceSafe[T]) Slice() []T {
	u.locker.RLock()
	defer u.locker.RUnlock()
	x := make([]T, 0, len(u.data))
	for k := range u.data {
		x = append(x, k)
	}
	return x
}
func (u *UniqueSliceSafe[T]) Has(item T) bool {
	u.locker.RLock()
	defer u.locker.RUnlock()
	for k := range u.data {
		if k == item {
			return true
		}
	}
	return false
}
