package queue

import (
	"container/list"
	"sync"
)

// Queue queue for go
type Queue struct {
	q      *list.List
	locker *sync.Mutex
}

// New get a new queue
func New() *Queue {
	mq := &Queue{
		q:      list.New(),
		locker: &sync.Mutex{},
	}
	return mq
}

// Clear clear queue list
func (mq *Queue) Clear() {
	mq.q.Init()
}

// Put put data to the end of the queue
func (mq *Queue) Put(value interface{}) {
	mq.locker.Lock()
	mq.q.PushBack(value)
	mq.locker.Unlock()
}

// PutFront put data to the first of the queue
func (mq *Queue) PutFront(value interface{}) {
	mq.locker.Lock()
	mq.q.PushFront(value)
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
func (mq *Queue) Len() int64 {
	return int64(mq.q.Len())
}

// Empty check if empty
func (mq *Queue) Empty() bool {
	return mq.q.Len() == 0
}
