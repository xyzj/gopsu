package mapfx

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// 使用示例：
// type sample struct {
// 	a string
// }
//
// var z = NewStructMap[string,sample]()
// z.Store("a", &sample{
// 	a: "132313",
// })

// StructMapI 结构体接口
//
//	这里不用any,因为any需要在代码中指定类型，使用泛型，在编译时会检查类型
type StructMapI interface{}

// NewStructMap 返回一个线程安全的基于基本数据类型的map,key为string,value为StructMapI 类型的struct
func NewStructMap[KEY int | int64 | uint64 | string, VALUE StructMapI]() *StructMap[KEY, VALUE] {
	return &StructMap[KEY, VALUE]{
		locker: sync.RWMutex{},
		data:   make(map[KEY]*VALUE),
	}
}

// StructMap 泛型map 对应各种slice类型
type StructMap[KEY int | int64 | uint64 | string, VALUE StructMapI] struct {
	locker sync.RWMutex
	data   map[KEY]*VALUE
}

// Store 添加内容
func (m *StructMap[KEY, VALUE]) Store(key KEY, value *VALUE) {
	m.locker.Lock()
	m.data[key] = value
	m.locker.Unlock()
}

// Delete 删除内容
func (m *StructMap[KEY, VALUE]) Delete(key KEY) {
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}

// Clean 清空内容
func (m *StructMap[KEY, VALUE]) Clean() {
	m.locker.Lock()
	m.data = make(map[KEY]*VALUE)
	m.locker.Unlock()
}

// Len 获取长度
func (m *StructMap[KEY, VALUE]) Len() int {
	m.locker.RLock()
	l := len(m.data)
	m.locker.RUnlock()
	return l
}

// Load 深拷贝一个值
//
//	获取的值可以安全编辑
func (m *StructMap[KEY, VALUE]) Load(key KEY) (*VALUE, bool) {
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
func (m *StructMap[KEY, VALUE]) LoadForUpdate(key KEY) (*VALUE, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		return v, true
	}
	return nil, false
}

// Has 判断Key是否存在
func (m *StructMap[KEY, VALUE]) Has(key KEY) bool {
	m.locker.RLock()
	defer m.locker.RUnlock()
	if _, ok := m.data[key]; ok {
		return true
	}
	return false
}

// HasPrefix 模糊判断Key是否存在
func (m *StructMap[KEY, VALUE]) HasPrefix(key string) bool {
	if key == "" {
		return false
	}
	m.locker.RLock()
	defer m.locker.RUnlock()
	ok := false
	for k := range m.data {
		if strings.HasPrefix(fmt.Sprintf("%v", k), key) {
			ok = true
			break
		}
	}
	return ok
}

// Clone 深拷贝map,可安全编辑
func (m *StructMap[KEY, VALUE]) Clone() map[KEY]*VALUE {
	x := make(map[KEY]*VALUE)
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
func (m *StructMap[KEY, VALUE]) ForEach(f func(key KEY, value *VALUE) bool) {
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
