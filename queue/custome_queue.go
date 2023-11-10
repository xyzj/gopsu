/*
Package queue 队列模块
HLQueue： wlst tcp module 定制模块，内部分为高低优先级2个队列，高优先级队列可设置先进先出或先进后出，低优先级队列只能先进先出。当高优先级队列有数据时，优先读取高优先级队列的数据
Queue： 常规队列模块
*/
package queue

import (
	"container/list"
	"sync"
	"sync/atomic"
)

// HLQueue 高低优先级队列，高优先级队列有数据时优先提取高优先级数据
type HLQueue struct {
	c      atomic.Int64
	locker sync.RWMutex
	high   *list.List
	low    *list.List
	fifo   bool // 是否先进先出
}

func (q *HLQueue) store(v interface{}, high bool) {
	q.locker.Lock()
	if high {
		if q.fifo {
			q.high.PushBack(v)
		} else {
			q.high.PushFront(v)
		}
	} else {
		q.low.PushBack(v)
	}
	q.c.Add(1)
	q.locker.Unlock()
}
func (q *HLQueue) load() (interface{}, bool) {
	q.locker.Lock()
	defer func() {
		q.locker.Unlock()
		q.c.Add(-1)
	}()
	if q.high.Len() > 0 {
		return q.high.Remove(q.high.Front()), true
	}
	if q.low.Len() > 0 {
		return q.low.Remove(q.low.Front()), true
	}
	return nil, false
}

// Clear 清空所有队列
func (q *HLQueue) Clear() {
	q.locker.Lock()
	q.high.Init()
	q.low.Init()
	q.c.Store(0)
	q.locker.Unlock()
}

// Len 返回队列长度，2个队列的总和
func (q *HLQueue) Len() int64 {
	return q.c.Load()
	// var l int64
	// q.locker.RLock()
	// l += int64(q.high.Len())
	// l += int64(q.low.Len())
	// q.locker.RUnlock()
	// return l
}

// Empty 队列是否为空
func (q *HLQueue) Empty() bool {
	return q.c.Load() == 0
}

// Put 添加低优先级数据
func (q *HLQueue) Put(v interface{}) {
	q.store(v, false)
}

// PutFront 添加高优先级数据
func (q *HLQueue) PutFront(v interface{}) {
	q.store(v, true)
}

// Get 获取数据
func (q *HLQueue) Get() (interface{}, bool) {
	return q.load()
}

// NewHLQueue 创建一个高低优先级队列，低优先级队列默认先进先出
//
//	fifo: 指定高优先级队列的进出方式，true-先进先出，false-先进后出
func NewHLQueue(fifo bool) *HLQueue {
	return &HLQueue{
		c:      atomic.Int64{},
		locker: sync.RWMutex{},
		high:   list.New(),
		low:    list.New(),
		fifo:   fifo,
	}
}
