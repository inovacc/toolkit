package ds

import "errors"

type RingBuffer[T any] interface {
	Push(value T)
	Pop() (T, bool)
	Peek() (T, bool)
	Len() int
	Cap() int
}

type ringBuffer[T any] struct {
	data        []T
	start, end  int
	size, limit int
}

func NewRingBuffer[T any](cap int) (*ringBuffer[T], error) {
	if cap <= 0 {
		return nil, errors.New("ring buffer capacity must be positive")
	}
	return &ringBuffer[T]{data: make([]T, cap), limit: cap}, nil
}

func (r *ringBuffer[T]) Push(val T) {
	if r.size == r.limit {
		r.start = (r.start + 1) % r.limit
	}
	r.data[r.end] = val
	r.end = (r.end + 1) % r.limit
	if r.size < r.limit {
		r.size++
	}
}

func (r *ringBuffer[T]) Pop() (T, bool) {
	if r.size == 0 {
		var zero T
		return zero, false
	}
	val := r.data[r.start]
	r.start = (r.start + 1) % r.limit
	r.size--
	return val, true
}

func (r *ringBuffer[T]) Peek() (T, bool) {
	if r.size == 0 {
		var zero T
		return zero, false
	}
	return r.data[r.start], true
}

func (r *ringBuffer[T]) Len() int { return r.size }
func (r *ringBuffer[T]) Cap() int { return r.limit }
