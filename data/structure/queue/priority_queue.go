package ds

import "container/heap"

// PriorityQueue is a generic interface for a priority-based queue where elements
// are dequeued based on their priority values.
type PriorityQueue[T any] interface {
	// Push adds an item with the given priority to the queue.
	// Items with lower priority values are dequeued first.
	Push(item T, priority int)

	// Pop removes and returns the highest priority item (lowest priority value).
	// Returns the value and true if successful, zero value and false if empty.
	Pop() (T, bool)

	// Peek returns the highest priority item without removing it.
	// Returns the value and true if successful, zero value and false if empty.
	Peek() (T, bool)

	// Len returns the number of elements in the queue.
	Len() int
}

type item[T any] struct {
	value    T
	priority int
	sequence int
	index    int
}

type minHeap[T any] []*item[T]

func (h *minHeap[T]) Len() int { return len(*h) }
func (h *minHeap[T]) Less(i, j int) bool {
	if (*h)[i].priority == (*h)[j].priority {
		return (*h)[i].sequence < (*h)[j].sequence
	}
	return (*h)[i].priority < (*h)[j].priority
}
func (h *minHeap[T]) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
	(*h)[i].index = i
	(*h)[j].index = j
}
func (h *minHeap[T]) Push(x any) { *h = append(*h, x.(*item[T])) }
func (h *minHeap[T]) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// HeapQueue implements PriorityQueue using a min-heap data structure.
// Items are dequeued in order of priority (lowest to highest) with FIFO ordering
// for items of equal priority.
type HeapQueue[T any] struct {
	queue    minHeap[T]
	sequence int
}

// NewPriorityQueue creates and returns a new empty HeapQueue.
func NewPriorityQueue[T any]() *HeapQueue[T] {
	hq := &HeapQueue[T]{}
	heap.Init(&hq.queue)
	return hq
}

// Push adds an item to the queue with the specified priority.
// Items with lower priority values are dequeued before items with higher priority values.
// For items with equal priorities, the order of insertion is preserved (FIFO).
func (hq *HeapQueue[T]) Push(value T, priority int) {
	item := &item[T]{
		value:    value,
		priority: priority,
		sequence: hq.sequence,
	}
	heap.Push(&hq.queue, item)
	hq.sequence++
}

// Pop removes and returns the highest priority item (lowest priority value).
// If multiple items have the same priority, the one inserted first will be returned.
// Returns the value and true if successful, zero value and false if the queue is empty.
func (hq *HeapQueue[T]) Pop() (T, bool) {
	if hq.queue.Len() == 0 {
		var zero T
		return zero, false
	}
	it := heap.Pop(&hq.queue).(*item[T])
	return it.value, true
}

// Peek returns the highest priority item without removing it from the queue.
// If multiple items have the same priority, the one inserted first will be returned.
// Returns the value and true if successful, zero value and false if the queue is empty.
func (hq *HeapQueue[T]) Peek() (T, bool) {
	if hq.queue.Len() == 0 {
		var zero T
		return zero, false
	}
	return hq.queue[0].value, true
}

// Len returns the current number of elements in the queue.
func (hq *HeapQueue[T]) Len() int {
	return hq.queue.Len()
}
