package queue

import (
	"sync/atomic"
)

// NewHighLowQueue creates a new high-low queue
// with the given parameters maxQueueSize, defaultQueueSize is 100
func NewHighLowQueue[VALUE any](maxQueueSize int32) *HighLowQueue[VALUE] {
	if maxQueueSize <= 0 {
		maxQueueSize = 100
	}
	return &HighLowQueue[VALUE]{
		high:   make(chan VALUE, maxQueueSize),
		low:    make(chan VALUE, maxQueueSize),
		lh:     atomic.Int32{},
		ll:     atomic.Int32{},
		max:    maxQueueSize,
		closed: atomic.Bool{},
	}
}

// HighLowQueue 一个基于channel的高低优先级队列，先进先出，队列为空时阻塞
type HighLowQueue[VALUE any] struct {
	high   chan VALUE
	low    chan VALUE
	out    VALUE
	max    int32
	lh     atomic.Int32
	ll     atomic.Int32
	closed atomic.Bool
	ok     bool
}

// Open initializes the high-low queue by creating channels for high and low priority items,
// setting the closed flag to false, and initializing the counters for high and low priority items.
// It does not return any value.
func (q *HighLowQueue[VALUE]) Open() {
	q.high = make(chan VALUE, q.max)
	q.low = make(chan VALUE, q.max)
	q.closed.Store(false)
	q.lh.Store(0)
	q.ll.Store(0)
}

// Close closes the high-low queue by setting the closed flag to true and closing the channels for high and low priority items.
// After calling Close, no more items can be added to the queue and Get will return immediately with an error.
//
// Close does not wait for any pending items to be processed. It is safe to call Close multiple times.
//
// Close does not return any value.
func (q *HighLowQueue[VALUE]) Close() {
	q.closed.Store(true)
	close(q.high)
	close(q.low)
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
func (q *HighLowQueue[VALUE]) Closed() bool {
	return q.closed.Load()
}

// Len returns the total number of items currently in the high-low queue.
// It does not block and returns the current length of the queue.
//
// Len does not return any error.
func (q *HighLowQueue[VALUE]) Len() int32 {
	return q.lh.Load() + q.ll.Load()
}

// Empty checks if the high-low queue is empty.
// It returns true if the queue is empty (i.e., there are no items in the queue),
// and false otherwise.
//
// Empty does not block and returns the current emptiness of the queue.
//
// Empty does not return any error.
func (q *HighLowQueue[VALUE]) Empty() bool {
	return q.lh.Load()+q.ll.Load() == 0
}

// put adds an item to the high-low queue based on the priority level.
//
// Parameters:
// - high: A boolean indicating whether the item should be added to the high priority channel (true) or the low priority channel (false).
// - v: The value of the item to be added to the queue.
//
// Return:
// - An error if the queue is closed or if the maximum queue size is reached, nil otherwise.
func (q *HighLowQueue[VALUE]) put(high bool, v VALUE) error {
	if q.closed.Load() {
		return ErrClosed
	}
	if q.lh.Load()+q.ll.Load() >= q.max {
		return ErrFull
	}
	if high {
		q.high <- v
		q.lh.Add(1)
	} else {
		q.low <- v
		q.ll.Add(1)
	}
	return nil
}

// Put adds an item to the low priority queue.
//
// Parameters:
// - f: The value of the item to be added to the queue.
//
// Return:
// - An error if the queue is closed or if the maximum queue size is reached, nil otherwise.
func (q *HighLowQueue[VALUE]) Put(f VALUE) error {
	return q.put(false, f)
}

// PutFront adds an item to the high priority queue.
// This function is designed to add an item to the front of the queue,
// ensuring that it is processed before items added with the Put function.
//
// Parameters:
//   - f: The value of the item to be added to the queue. This parameter can be of any type
//     that matches the generic type VALUE specified when creating the HighLowQueue instance.
//
// Return:
//   - An error if the queue is closed or if the maximum queue size is reached.
//     In such cases, the function returns a non-nil error.
//   - nil if the item is successfully added to the queue.
func (q *HighLowQueue[VALUE]) PutFront(f VALUE) error {
	return q.put(true, f)
}

// Get retrieves and removes an item from the high-low queue based on priority.
// If there are items in the high priority channel, it retrieves and removes an item from there.
// If there are no items in the high priority channel, it retrieves and removes an item from the low priority channel.
//
// Return:
//   - VALUE: The value of the retrieved item. The type of VALUE is determined by the generic type specified when creating the HighLowQueue instance.
//   - bool: A boolean indicating whether the retrieval was successful.
//   - true: The retrieval was successful and an item was returned.
//   - false: The retrieval failed because the queue is closed or empty.
func (q *HighLowQueue[VALUE]) Get() (VALUE, bool) {
	if q.closed.Load() {
		return *new(VALUE), false
	}
	if q.lh.Load() > 0 {
		q.out, q.ok = <-q.high
		q.lh.Add(-1)
		return q.out, q.ok
	}

	select {
	case q.out, q.ok = <-q.high:
		q.lh.Add(-1)
	case q.out, q.ok = <-q.low:
		q.ll.Add(-1)
	}
	return q.out, q.ok
}
