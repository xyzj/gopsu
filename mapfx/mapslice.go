package mapfx

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// 使用示例：
// var a = NewSliceMap[int64]()
// a.Store("a",[]int64{1})
// -- a["a"]=[]int64{1}
// a.StoreItem("a",int64(1)) // 重复值不会被添加
// a.StoreItem("a",int64(2))
// a.StoreItem("a",int64(3))
// -- a["a"]=[]int64{1,2,3}

// NewSliceMap 返回一个线程安全的基于基本数据类型的map,key为string,value为slice
func NewSliceMap[T any]() *SliceMap[T] {
	return &SliceMap[T]{
		locker: sync.RWMutex{},
		data:   make(map[string][]T),
	}
}

// SliceMap 泛型map 对应各种slice类型
type SliceMap[T any] struct {
	locker sync.RWMutex
	data   map[string][]T
}

// Store 添加内容
func (m *SliceMap[T]) Store(key string, value []T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	m.data[key] = value
	m.locker.Unlock()
}

// StoreItem 向指定的key的切片内添加一个值，重复值不会添加
func (m *SliceMap[T]) StoreItem(key string, item T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	defer m.locker.Unlock()
	if v, ok := m.data[key]; ok {
		for _, vv := range v {
			if reflect.DeepEqual(vv, item) {
				return
			}
		}
		m.data[key] = append(m.data[key], item)
	} else {
		m.data[key] = []T{item}
	}
}

// Delete 删除内容
func (m *SliceMap[T]) Delete(key string) {
	if key == "" {
		return
	}
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}

// DeleteItem 删除一个值
func (m *SliceMap[T]) DeleteItem(key string, item T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	defer m.locker.Unlock()
	if v, ok := m.data[key]; ok {
		for k, vv := range v {
			if reflect.DeepEqual(vv, item) {
				m.data[key] = append(v[:k], v[k+1:]...)
				return
			}
		}
	}
}

// Clean 清空内容
func (m *SliceMap[T]) Clean() {
	m.locker.Lock()
	for k := range m.data {
		delete(m.data, k)
	}
	// m.data = make(map[string][]T)
	m.locker.Unlock()
}

// Len 获取长度
func (m *SliceMap[T]) Len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}

// Load 读取一个值
func (m *SliceMap[T]) Load(key string) ([]T, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		return v, true
	}
	return v, false
}

// Has 判断Key是否存在
func (m *SliceMap[T]) Has(key string) bool {
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

// HasPrefix 模糊判断Key是否存在
func (m *SliceMap[T]) HasPrefix(key string) bool {
	if key == "" {
		return false
	}
	m.locker.RLock()
	defer m.locker.RUnlock()
	ok := false
	for k := range m.data {
		if strings.HasPrefix(k, key) {
			ok = true
			break
		}
	}
	return ok
}

// HasItem 判断Key-item是否存在
func (m *SliceMap[T]) HasItem(key string, item T) bool {
	if key == "" {
		return false
	}
	m.locker.RLock()
	defer m.locker.RUnlock()
	if v, ok := m.data[key]; ok {
		for _, vv := range v {
			if reflect.DeepEqual(vv, item) {
				return true
			}
		}
	}
	return false
}

// Clone 深拷贝map,可安全编辑
func (m *SliceMap[T]) Clone() map[string][]T {
	x := make(map[string][]T)
	m.locker.RLock()
	for k, v := range m.data {
		x[k] = v
	}
	m.locker.RUnlock()
	return x
}

// ForEach 遍历map的key和value
func (m *SliceMap[T]) ForEach(f func(key string, value []T) bool) (err error) {
	x := m.Clone()
	defer func() {
		if ex := recover(); ex != nil {
			err = errors.WithStack(ex.(error))
			println(fmt.Sprintf("map foreach error :%+v", errors.WithStack(err)))
		}
	}()
	for k, v := range x {
		if !f(k, v) {
			break
		}
	}
	return err
}

// Keys 返回所有Key
func (m *SliceMap[T]) Keys() []string {
	m.locker.RLock()
	defer m.locker.RUnlock()
	return Keys[string](m.data)
}
