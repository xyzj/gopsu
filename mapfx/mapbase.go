package mapfx

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// 使用示例：
// var a = NewBaseMap[string]()
// a.Store("a","1")

// NewBaseMap 返回一个线程安全的基于基本数据类型的map,key为string,value为基本数据类型，支持空结构
//
//	value类型支持 byte | int8 | int | int32 | int64 | float32 | float64 | string | struct{}
//
//	struct{} 用于空类型，用在使用map去重的场景，可以大大降低内存分配
func NewBaseMap[T byte | int8 | int | int32 | int64 | float32 | float64 | string | struct{}]() *BaseMap[T] {
	return &BaseMap[T]{
		locker: sync.RWMutex{},
		data:   make(map[string]T),
	}
}

// BaseMap 泛型map 对应各种基础类型
type BaseMap[T byte | int8 | int | int32 | int64 | float32 | float64 | string | struct{}] struct {
	locker sync.RWMutex
	data   map[string]T
}

// Store 添加内容
func (m *BaseMap[T]) Store(key string, value T) {
	if key == "" {
		return
	}
	m.locker.Lock()
	m.data[key] = value
	m.locker.Unlock()
}

// Delete 删除内容
func (m *BaseMap[T]) Delete(key string) {
	if key == "" {
		return
	}
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}

// Clean 清空内容
func (m *BaseMap[T]) Clean() {
	m.locker.Lock()
	for k := range m.data {
		delete(m.data, k)
	}
	m.locker.Unlock()
}

// Len 获取长度
func (m *BaseMap[T]) Len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}

// Load 深拷贝一个值
func (m *BaseMap[T]) Load(key string) (T, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		return v, true
	}
	return v, false
}

// Has 判断Key是否存在
func (m *BaseMap[T]) Has(key string) bool {
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
func (m *BaseMap[T]) HasPrefix(key string) bool {
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

// Clone 深拷贝map,可安全编辑
func (m *BaseMap[T]) Clone() map[string]T {
	x := make(map[string]T)
	m.locker.RLock()
	for k, v := range m.data {
		x[k] = v
	}
	m.locker.RUnlock()
	return x
}

// ForEach 遍历map的key和value
func (m *BaseMap[T]) ForEach(f func(key string, value T) bool) {
	x := m.Clone()
	defer func() {
		if err := recover(); err != nil {
			println(fmt.Sprintf("%+v", err))
		}
	}()
	for k, v := range x {
		if !f(k, v) {
			break
		}
	}
}

// ToJSON 返回json字符串
func (m *BaseMap[T]) ToJSON() []byte {
	m.locker.RLock()
	defer m.locker.RUnlock()
	b, err := json.Marshal(m.data)
	if err != nil {
		return []byte{}
	}
	return b
}

// FromJSON 从json字符串初始化数据
func (m *BaseMap[T]) FromJSON(b []byte) error {
	m.locker.Lock()
	defer m.locker.Unlock()
	return json.Unmarshal(b, &m.data)
}

// FromFile 从json字符串初始化数据
func (m *BaseMap[T]) FromFile(f string) error {
	b, err := os.ReadFile(f)
	if err != nil {
		return err
	}
	return m.FromJSON(b)
}

// ToFile 保存到文件
func (m *BaseMap[T]) ToFile(f string) error {
	m.locker.Lock()
	defer m.locker.Unlock()
	b, err := json.Marshal(m.data)
	if err != nil {
		return err
	}
	return os.WriteFile(f, b, 0664)
}
