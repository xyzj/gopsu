package queue

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type queueItem[VALUE any] struct {
	priority string
	data     VALUE
}

// PriorityQueue 一个可以设置内容优先级的队列，可设置在同等优先级下的内容先入先出或后入先出，队列为空时不阻塞
type PriorityQueue[VALUE any] struct {
	locker sync.RWMutex
	data   []*queueItem[VALUE]
	item   *queueItem[VALUE]
	l      atomic.Int32
	closed atomic.Bool
	max    int32
	fifo   bool
}

// NewPriorityQueue creates a new instance of PriorityQueue.
// The PriorityQueue is a generic type that supports any type of value.
//
// fifo: A boolean flag indicating whether the queue should follow a FIFO (First In First Out) order.
//
//	If true, items with lower priority values will be dequeued first.
//	If false, items with higher priority values will be dequeued first.
//
// Returns a pointer to a new PriorityQueue instance.
func NewPriorityQueue[VALUE any](maxQueueSize int32, fifo bool) *PriorityQueue[VALUE] {
	return &PriorityQueue[VALUE]{
		locker: sync.RWMutex{},
		data:   make([]*queueItem[VALUE], 0, maxQueueSize),
		max:    maxQueueSize,
		fifo:   fifo,
		l:      atomic.Int32{},
		closed: atomic.Bool{},
	}
}

// Open initializes the high-low queue by creating channels for high and low priority items,
// setting the closed flag to false, and initializing the counters for high and low priority items.
// It does not return any value.
func (q *PriorityQueue[VALUE]) Open() {
	q.closed.Store(false)
	q.l.Store(0)
	q.data = make([]*queueItem[VALUE], 0, q.max)
}

// Close closes the high-low queue by setting the closed flag to true and closing the channels for high and low priority items.
// After calling Close, no more items can be added to the queue and Get will return immediately with an error.
//
// Close does not wait for any pending items to be processed. It is safe to call Close multiple times.
//
// Close does not return any value.
func (q *PriorityQueue[VALUE]) Close() {
	q.closed.Store(true)
	q.l.Store(0)
}

// Closed checks if the high-low queue is closed.
//
// The Closed function returns true if the queue is closed (i.e., no more items can be added to the queue),
// and false otherwise. It does not block and returns the current state of the queue.
//
// The Closed function does not return any error.
//
// Return:
//   - bool: A boolean indicating whether the queue is closed.
//   - true: The queue is closed.
//   - false: The queue is open.
func (q *PriorityQueue[VALUE]) Closed() bool {
	return q.closed.Load()
}

// Len returns the number of items currently in the PriorityQueue.
//
// The Len function is a method of the PriorityQueue type. It returns the length of the data slice,
// which represents the number of items currently in the queue.
//
// Returns:
// - An integer representing the number of items in the PriorityQueue.
func (q *PriorityQueue[VALUE]) Len() int32 {
	return q.l.Load()
}

// Empty checks if the high-low queue is empty.
// It returns true if the queue is empty (i.e., there are no items in the queue),
// and false otherwise.
//
// Empty does not block and returns the current emptiness of the queue.
//
// Empty does not return any error.
func (q *PriorityQueue[VALUE]) Empty() bool {
	return q.l.Load() == 0
}

// Put adds an item to the PriorityQueue with the default priority (255).
// If the PriorityQueue is already full, it returns an error.
//
// Parameters:
// - data: The value to be added to the PriorityQueue.
//
// Returns:
// - error: An error indicating whether the item was successfully added (nil) or if the queue is full (ErrFull).
func (q *PriorityQueue[VALUE]) Put(data VALUE) error {
	return q.PutWithPriority(data, 255)
}

// PutFront adds an item to the PriorityQueue with the highest priority (0).
// If the PriorityQueue is already full, it returns an error.
//
// Parameters:
// - data: The value to be added to the PriorityQueue. The type of data must be compatible with the generic type VALUE.
//
// Returns:
// - error: An error indicating whether the item was successfully added (nil) or if the queue is full (ErrFull).
//
// PutFront is a method of the PriorityQueue type. It calls the PutWithPriority method with a priority of 0,
// effectively adding the item to the front of the queue.
func (q *PriorityQueue[VALUE]) PutFront(data VALUE) error {
	return q.PutWithPriority(data, 0)
}

// PutWithPriority adds an item to the PriorityQueue with the specified priority.
//
// The Put function locks the PriorityQueue to ensure thread safety while adding the item.
// It creates a new queueItem with the provided data and priority, and appends it to the data slice.
// After adding the item, it calls the sort function to maintain the correct order of items in the queue.
//
// Parameters:
// - data: The value to be added to the PriorityQueue.
// - priority: The priority of the item. Lower values indicate higher priority.
func (q *PriorityQueue[VALUE]) PutWithPriority(data VALUE, priority byte) error {
	if q.closed.Load() {
		return ErrClosed
	}
	if q.l.Load() >= q.max {
		return ErrFull
	}
	q.locker.Lock()
	defer q.locker.Unlock()
	q.l.Add(1)
	q.data = append(q.data, &queueItem[VALUE]{
		priority: fmt.Sprintf("%03d_%d", priority, time.Now().UnixNano()),
		data:     data,
	})
	q.sort()
	return nil
}

// Get retrieves and removes the highest priority item from the PriorityQueue.
// If the PriorityQueue is empty, it returns a zero value of the VALUE type and false.
//
// The function locks the PriorityQueue to ensure thread safety while retrieving the item.
// It checks if the queue is empty and returns false if it is.
// Otherwise, it assigns the first item in the queue to the item variable, removes it from the data slice,
// and returns the data value and true.
//
// Returns:
//
//	data VALUE - The data value of the highest priority item.
//	bool - A boolean indicating whether an item was retrieved (true) or the queue was empty (false).
func (q *PriorityQueue[VALUE]) Get() (VALUE, bool) {
	if q.l.Load() == 0 || q.closed.Load() {
		return *new(VALUE), false
	}
	q.locker.Lock()
	defer q.locker.Unlock()
	q.item = q.data[0]
	q.data = q.data[1:]
	q.l.Add(-1)
	return q.item.data, true
}

func (q *PriorityQueue[VALUE]) sort() {
	q.locker.Lock()
	defer q.locker.Unlock()
	sort.Slice(q.data, func(i, j int) bool {
		if q.fifo {
			return q.data[i].priority < q.data[j].priority
		}
		return q.data[i].priority > q.data[j].priority
	})
}
