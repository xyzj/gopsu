package queue

import "errors"

var (
	// ErrClosed is returned when trying to get an item from a closed queue.
	ErrClosed = errors.New("queue is closed")

	// ErrEmpty is returned when trying to remove an item from an empty queue.
	ErrEmpty = errors.New("queue is empty")

	// ErrFull is returned when trying to add an item to a full queue.
	ErrFull = errors.New("queue is full")
)
