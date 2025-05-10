// Package ds provides generic data structures for common use cases.
// This file implements a FIFO queue.
package ds

// Queue is a generic FIFO (First-In-First-Out) queue interface that can store
// elements of any type. It provides basic queue operations including enqueue and dequeue.
type Queue[T any] interface {
	// Enqueue adds a new value to the end of the queue.
	Enqueue(value T)

	// Dequeue removes and returns the first value from the queue.
	// Returns the value and true if successful, zero value and false if empty.
	Dequeue() (T, bool)

	// Peek returns the first value without removing it.
	// Returns the value and true if successful, zero value and false if empty.
	Peek() (T, bool)

	// Len returns the number of elements in the queue.
	Len() int

	// IsEmpty returns true if the queue has no elements.
	IsEmpty() bool
}

// SliceQueue is a slice-based implementation of the Queue interface.
type SliceQueue[T any] struct {
	items []T
}

// NewQueue returns an empty queue.
func NewQueue[T any]() *SliceQueue[T] {
	return &SliceQueue[T]{}
}

// Enqueue adds a new value to the end of the queue.
func (q *SliceQueue[T]) Enqueue(value T) {
	q.items = append(q.items, value)
}

// Dequeue removes and returns the first value from the queue.
func (q *SliceQueue[T]) Dequeue() (T, bool) {
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

// Peek returns the first value without removing it.
func (q *SliceQueue[T]) Peek() (T, bool) {
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	return q.items[0], true
}

// Len returns the number of elements in the queue.
func (q *SliceQueue[T]) Len() int { return len(q.items) }

// IsEmpty returns true if the queue has no elements.
func (q *SliceQueue[T]) IsEmpty() bool { return len(q.items) == 0 }
