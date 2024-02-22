// Package queue safety queue
package queue

import (
	"container/list"
	"sync"
	"sync/atomic"
)

// Queue queue for go
type Queue struct {
	c      atomic.Int32
	q      *list.List
	locker *sync.RWMutex
}

// New get a new queue
func New() *Queue {
	mq := &Queue{
		c:      atomic.Int32{},
		q:      list.New(),
		locker: &sync.RWMutex{},
	}
	return mq
}

// Clean clean queue list
func (mq *Queue) Clean() {
	mq.locker.Lock()
	mq.q.Init()
	mq.c.Store(0)
	mq.locker.Unlock()
}

// Put put data to the end of the queue
func (mq *Queue) Put(value interface{}) {
	mq.locker.Lock()
	mq.q.PushBack(value)
	mq.c.Add(1)
	mq.locker.Unlock()
}

// PutFront put data to the first of the queue
func (mq *Queue) PutFront(value interface{}) {
	mq.locker.Lock()
	mq.q.PushFront(value)
	mq.c.Add(1)
	mq.locker.Unlock()
}

// Get get data from front
func (mq *Queue) Get() interface{} {
	mq.locker.Lock()
	defer mq.locker.Unlock()
	if mq.q.Len() == 0 {
		return nil
	}
	e := mq.q.Remove(mq.q.Front())
	mq.c.Add(-1)
	return e
}

// GetNoDel get data from front
func (mq *Queue) GetNoDel() interface{} {
	mq.locker.Lock()
	defer mq.locker.Unlock()
	if mq.q.Len() == 0 {
		return nil
	}
	e := mq.q.Front().Value
	return e
}

// Len get queue len
func (mq *Queue) Len() int {
	return int(mq.c.Load())
	// mq.locker.RLock()
	// defer mq.locker.RUnlock()
	// return int64(mq.q.Len())
}

// Empty check if empty
func (mq *Queue) Empty() bool {
	return mq.c.Load() == 0
	// mq.locker.RLock()
	// defer mq.locker.RUnlock()
	// return mq.q.Len() == 0
}
