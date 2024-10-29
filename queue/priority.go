package queue

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type queueItem[VALUE any] struct {
	priority string
	data     VALUE
}

type PriorityQueue[VALUE any] struct {
	locker sync.RWMutex
	data   []*queueItem[VALUE]
	item   *queueItem[VALUE]
	max    int
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
func NewPriorityQueue[VALUE any](maxQueueSize int, fifo bool) *PriorityQueue[VALUE] {
	return &PriorityQueue[VALUE]{
		locker: sync.RWMutex{},
		data:   make([]*queueItem[VALUE], 0, maxQueueSize),
		max:    maxQueueSize,
		fifo:   fifo,
	}
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
func (p *PriorityQueue[VALUE]) Get() (VALUE, bool) {
	p.locker.Lock()
	defer p.locker.Unlock()
	if len(p.data) == 0 {
		return *new(VALUE), false
	}
	p.item = p.data[0]
	p.data = p.data[1:]
	return p.item.data, true
}

// Put adds an item to the PriorityQueue with the specified priority.
//
// The Put function locks the PriorityQueue to ensure thread safety while adding the item.
// It creates a new queueItem with the provided data and priority, and appends it to the data slice.
// After adding the item, it calls the sort function to maintain the correct order of items in the queue.
//
// Parameters:
// - data: The value to be added to the PriorityQueue.
// - priority: The priority of the item. Lower values indicate higher priority.
func (p *PriorityQueue[VALUE]) Put(data VALUE, priority byte) error {
	p.locker.Lock()
	defer p.locker.Unlock()
	if len(p.data) >= p.max {
		return ErrFull
	}
	p.data = append(p.data, &queueItem[VALUE]{
		priority: fmt.Sprintf("%03d_%d", priority, time.Now().UnixNano()),
		data:     data,
	})
	p.sort()
	return nil
}

// Len returns the number of items currently in the PriorityQueue.
//
// The Len function is a method of the PriorityQueue type. It returns the length of the data slice,
// which represents the number of items currently in the queue.
//
// Returns:
// - An integer representing the number of items in the PriorityQueue.
func (p *PriorityQueue[VALUE]) Len() int {
	p.locker.RLock()
	defer p.locker.RUnlock()
	return len(p.data)
}

func (p *PriorityQueue[VALUE]) sort() {
	p.locker.Lock()
	defer p.locker.Unlock()
	sort.Slice(p.data, func(i, j int) bool {
		if p.fifo {
			return p.data[i].priority < p.data[j].priority
		}
		return p.data[i].priority > p.data[j].priority
	})
}
