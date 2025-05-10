// Package ds provides generic data structure implementations including stacks, queues,
// sets, priority queues, and linked lists. These implementations are designed to be
// type-safe and reusable across different data types using Go's generics.
package ds

// Stack is a generic LIFO (Last-In-First-Out) stack interface that can store elements
// of any type. It provides basic stack operations including push, pop, and peek.
type Stack[T any] interface {
	// Push adds a new value to the top of the stack.
	Push(value T)

	// Pop removes and returns the top value of the stack.
	// Returns the value and true if successful, zero value and false if empty.
	Pop() (T, bool)

	// Peek returns the top value without removing it.
	// Returns the value and true if successful, zero value and false if empty.
	Peek() (T, bool)

	// Len returns the number of elements in the stack.
	Len() int

	// IsEmpty returns true if the stack has no elements.
	IsEmpty() bool
}

// SliceStack is a slice-based implementation of the Stack interface.
// It provides an efficient, dynamic-size stack implementation.
type SliceStack[T any] struct {
	items []T
}

// NewStack creates and returns an empty stack.
func NewStack[T any]() *SliceStack[T] {
	return &SliceStack[T]{}
}

// Push adds a new value to the top of the stack.
func (s *SliceStack[T]) Push(value T) {
	s.items = append(s.items, value)
}

// Pop removes and returns the top value of the stack.
func (s *SliceStack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	val := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return val, true
}

// Peek returns the top value without removing it.
func (s *SliceStack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

// Len returns the number of elements in the stack.
func (s *SliceStack[T]) Len() int { return len(s.items) }

// IsEmpty returns true if the stack has no elements.
func (s *SliceStack[T]) IsEmpty() bool { return len(s.items) == 0 }
