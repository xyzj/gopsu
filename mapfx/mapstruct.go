package mapfx

import (
	"sync"

	"github.com/pkg/errors"
)

// StructMapI 结构体接口
//
//	just add an empty func named DoNothing()
type StructMapI interface {
	DoNoting()
}

// NewStructMap 返回一个线程安全的基于基本数据类型的map,key为string,value为StructMapI 类型的struct
//
//	StructMapI 只需要一个DoNothing()方法，不需要做任何事情
func NewStructMap[T StructMapI]() *StructMap[T] {
	return &StructMap[T]{
		locker: sync.RWMutex{},
		data:   make(map[string]*T),
	}
}

// StructMap 泛型map 对应各种slice类型
type StructMap[T StructMapI] struct {
	locker sync.RWMutex
	data   map[string]*T
}

// Store 添加内容
func (m *StructMap[T]) Store(key string, value *T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	m.data[key] = value
	m.locker.Unlock()
}

// Delete 删除内容
func (m *StructMap[T]) Delete(key string) {
	if key == "" {
		return
	}
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}

// Clean 清空内容
func (m *StructMap[T]) Clean() {
	m.locker.Lock()
	m.data = make(map[string]*T)
	m.locker.Unlock()
}

// Len 获取长度
func (m *StructMap[T]) Len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}

// Load 深拷贝一个值
//
//	获取的值可以安全编辑
func (m *StructMap[T]) Load(key string) (*T, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		z := *v
		return &z, true
	}
	return nil, false
}

// LoadForUpdate 浅拷贝一个值
//
//	可用于需要直接修改map内的值的场景，会引起map内值的变化
func (m *StructMap[T]) LoadForUpdate(key string) (*T, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		return v, true
	}
	return nil, false
}

// Has 判断Key是否存在
func (m *StructMap[T]) Has(key string) bool {
	if key == "" {
		return false
	}
	m.locker.RLock()
	defer m.locker.RUnlock()
	if _, ok := m.data[key]; ok {
		return true
	}
	return false
}

// Clone 深拷贝map,可安全编辑
func (m *StructMap[T]) Clone() map[string]*T {
	x := make(map[string]*T)
	m.locker.RLock()
	for k, v := range m.data {
		z := *v
		x[k] = &z
	}
	m.locker.RUnlock()
	return x
}

// ForEach 遍历map的key和value
//
//	遍历前会进行深拷贝，可安全编辑
func (m *StructMap[T]) ForEach(f func(key string, value *T) bool) {
	x := m.Clone()
	defer func() {
		if err := recover(); err != nil {
			println(errors.WithStack(err.(error)).Error())
		}
	}()
	for k, v := range x {
		if !f(k, v) {
			break
		}
	}
}
