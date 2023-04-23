package mapfx

import (
	"sync"
)

// NewBaseMap 返回一个线程安全的基于基本数据类型的map,key为string,value为基本数据类型，支持空结构
//
// value类型支持 byte | int8 | int | int32 | int64 | float32 | float64 | string
func NewBaseMap[T byte | int8 | int | int32 | int64 | float32 | float64 | string | struct{}]() *BaseMap[T] {
	return &BaseMap[T]{
		locker: sync.RWMutex{},
		data:   make(map[string]T),
	}
}

// BaseMap 泛型map 对应各种slice类型
type BaseMap[T byte | int8 | int | int32 | int64 | float32 | float64 | string | struct{}] struct {
	locker sync.RWMutex
	data   map[string]T
}

func (m *BaseMap[T]) Store(key string, value T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	m.data[key] = value
	m.locker.Unlock()
}
func (m *BaseMap[T]) Delete(key string) {
	if key == "" {
		return
	}
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}
func (m *BaseMap[T]) Clean() {
	m.locker.Lock()
	m.data = make(map[string]T)
	m.locker.Unlock()
}
func (m *BaseMap[T]) Len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}
func (m *BaseMap[T]) Load(key string) (T, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		return v, true
	}
	return v, false
}
func (m *BaseMap[T]) Clone() map[string]T {
	x := make(map[string]T)
	m.locker.RLock()
	for k, v := range m.data {
		x[k] = v
	}
	m.locker.RUnlock()
	return x
}
func (m *BaseMap[T]) ForEach(f func(key string, value T) bool) {
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
