package ds

import (
	"testing"
)

func TestSliceQueue(t *testing.T) {
	t.Run("new queue should be empty", func(t *testing.T) {
		queue := NewQueue[int]()
		if !queue.IsEmpty() {
			t.Error("new queue should be empty")
		}
		if queue.Len() != 0 {
			t.Errorf("expected length 0, got %d", queue.Len())
		}

		// Verify Dequeue and Peek return false on empty queue
		_, ok := queue.Dequeue()
		if ok {
			t.Error("Dequeue on empty queue should return false")
		}

		_, ok = queue.Peek()
		if ok {
			t.Error("Peek on empty queue should return false")
		}
	})

	t.Run("FIFO ordering", func(t *testing.T) {
		queue := NewQueue[string]()

		// Test enqueue operations
		values := []string{"first", "second", "third"}
		for _, v := range values {
			queue.Enqueue(v)
		}

		if queue.Len() != 3 {
			t.Errorf("expected length 3, got %d", queue.Len())
		}

		if queue.IsEmpty() {
			t.Error("queue should not be empty after enqueuing")
		}

		// Test peek operation
		val, ok := queue.Peek()
		if !ok {
			t.Error("Peek should return true for non-empty queue")
		}
		if val != "first" {
			t.Errorf("expected Peek() to return 'first', got %v", val)
		}

		// Verify FIFO order through dequeue
		for i, expected := range values {
			val, ok := queue.Dequeue()
			if !ok {
				t.Errorf("Dequeue at index %d should return true", i)
			}
			if val != expected {
				t.Errorf("expected %s, got %s", expected, val)
			}
		}

		// Verify queue is empty after all dequeues
		if !queue.IsEmpty() {
			t.Error("queue should be empty after dequeuing all elements")
		}
	})

	t.Run("mixed operations", func(t *testing.T) {
		queue := NewQueue[int]()

		// Initial enqueue
		queue.Enqueue(1)
		queue.Enqueue(2)

		// Verify first element
		val, ok := queue.Peek()
		if !ok || val != 1 {
			t.Errorf("expected Peek() to return 1, got %v", val)
		}

		// Dequeue one element
		val, ok = queue.Dequeue()
		if !ok || val != 1 {
			t.Errorf("expected Dequeue() to return 1, got %v", val)
		}

		// Add more elements
		queue.Enqueue(3)
		queue.Enqueue(4)

		// Verify remaining elements
		expected := []int{2, 3, 4}
		for i, exp := range expected {
			val, ok := queue.Dequeue()
			if !ok {
				t.Errorf("Dequeue at index %d should return true", i)
			}
			if val != exp {
				t.Errorf("expected %d, got %d", exp, val)
			}
		}
	})

	t.Run("large queue operations", func(t *testing.T) {
		queue := NewQueue[int]()

		// Enqueue 1000 items
		for i := 0; i < 1000; i++ {
			queue.Enqueue(i)
		}

		if queue.Len() != 1000 {
			t.Errorf("expected length 1000, got %d", queue.Len())
		}

		// Dequeue all items and verify order
		for i := 0; i < 1000; i++ {
			val, ok := queue.Dequeue()
			if !ok {
				t.Errorf("Dequeue at index %d should return true", i)
			}
			if val != i {
				t.Errorf("expected %d, got %d", i, val)
			}
		}

		if !queue.IsEmpty() {
			t.Error("queue should be empty after dequeuing all elements")
		}
	})
}
