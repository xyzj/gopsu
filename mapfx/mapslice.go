package mapfx

import (
	"sync"
)

// NewSliceMap 返回一个线程安全的基于基本数据类型的map,key为string,value为slice
//
// value类型支持 byte | int8 | int | int32 | int64 | float32 | float64 | string
func NewSliceMap[T byte | int8 | int | int32 | int64 | float32 | float64 | string]() *SliceMap[T] {
	return &SliceMap[T]{
		locker: sync.RWMutex{},
		data:   make(map[string][]T),
	}
}

// SliceMap 泛型map 对应各种slice类型
type SliceMap[T byte | int8 | int | int32 | int64 | float32 | float64 | string] struct {
	locker sync.RWMutex
	data   map[string][]T
}

func (m *SliceMap[T]) Store(key string, value []T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	m.data[key] = value
	m.locker.Unlock()
}
func (m *SliceMap[T]) StoreItem(key string, item T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	defer m.locker.Unlock()
	if v, ok := m.data[key]; ok {
		for _, vv := range v {
			if vv == item { // 已有，不处理
				return
			}
		}
		m.data[key] = append(m.data[key], item)
	} else {
		m.data[key] = []T{item}
	}
}
func (m *SliceMap[T]) Delete(key string) {
	if key == "" {
		return
	}
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}
func (m *SliceMap[T]) Clean() {
	m.locker.Lock()
	m.data = make(map[string][]T)
	m.locker.Unlock()
}
func (m *SliceMap[T]) Len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}
func (m *SliceMap[T]) Load(key string) ([]T, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		return v, true
	}
	return v, false
}
func (m *SliceMap[T]) Has(key string, item any) bool {
	if key == "" {
		return false
	}
	m.locker.RLock()
	defer m.locker.RUnlock()
	if v, ok := m.data[key]; ok {
		for _, vv := range v {
			if vv == item {
				return true
			}
		}
	}
	return false
}
func (m *SliceMap[T]) Clone() map[string][]T {
	x := make(map[string][]T)
	m.locker.RLock()
	for k, v := range m.data {
		x[k] = v
	}
	m.locker.RUnlock()
	return x
}
func (m *SliceMap[T]) ForEach(f func(key string, value []T) bool) {
	x := m.Clone()
	defer func() {
		if err := recover(); err != nil {
			println(err.(error).Error())
		}
	}()
	for k, v := range x {
		if !f(k, v) {
			break
		}
	}
}
