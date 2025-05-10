package ds

import (
	"testing"
)

func TestHeapQueue(t *testing.T) {
	t.Run("new queue should be empty", func(t *testing.T) {
		queue := NewPriorityQueue[string]()
		if queue.Len() != 0 {
			t.Errorf("expected length 0, got %d", queue.Len())
		}

		// Verify Pop and Peek return false on empty queue
		_, ok := queue.Pop()
		if ok {
			t.Error("Pop on empty queue should return false")
		}

		_, ok = queue.Peek()
		if ok {
			t.Error("Peek on empty queue should return false")
		}
	})

	t.Run("priority ordering", func(t *testing.T) {
		queue := NewPriorityQueue[string]()

		// Insert items with different priorities
		queue.Push("low", 3)
		queue.Push("medium", 2)
		queue.Push("high", 1)

		if queue.Len() != 3 {
			t.Errorf("expected length 3, got %d", queue.Len())
		}

		// Verify peek shows the highest priority (the lowest number) item
		val, ok := queue.Peek()
		if !ok || val != "high" {
			t.Errorf("expected Peek() to return 'high', got %v", val)
		}

		// Verify items are popped in the priority order
		expected := []string{"high", "medium", "low"}
		for i, exp := range expected {
			val, ok := queue.Pop()
			if !ok {
				t.Errorf("Pop() at index %d should return true", i)
			}
			if val != exp {
				t.Errorf("expected %s, got %s", exp, val)
			}
		}

		// Verify the queue is empty after all pops
		if queue.Len() != 0 {
			t.Errorf("expected empty queue, got length %d", queue.Len())
		}
	})

	t.Run("same priority handling", func(t *testing.T) {
		queue := NewPriorityQueue[int]()

		// Insert items with the same priority
		queue.Push(1, 1)
		queue.Push(2, 1)
		queue.Push(3, 1)

		// Verify items with same priority maintain FIFO order
		for i := 1; i <= 3; i++ {
			val, ok := queue.Pop()
			if !ok {
				t.Errorf("Pop() at index %d should return true", i)
			}
			if val != i {
				t.Errorf("expected %d, got %d", i, val)
			}
		}
	})

	t.Run("mixed operations", func(t *testing.T) {
		queue := NewPriorityQueue[int]()

		// Perform mixed operations
		queue.Push(1, 3)
		queue.Push(2, 1)
		queue.Push(3, 2)

		// Verify first peek
		val, ok := queue.Peek()
		if !ok || val != 2 {
			t.Errorf("expected Peek() to return 2, got %v", val)
		}

		// Pop the highest priority
		val, ok = queue.Pop()
		if !ok || val != 2 {
			t.Errorf("expected Pop() to return 2, got %v", val)
		}

		// Add the new highest priority
		queue.Push(4, 0)

		// Verify new order
		expected := []int{4, 3, 1}
		for i, exp := range expected {
			val, ok := queue.Pop()
			if !ok {
				t.Errorf("Pop() at index %d should return true", i)
			}
			if val != exp {
				t.Errorf("expected %d, got %d", exp, val)
			}
		}
	})

	t.Run("stress test with multiple priorities", func(t *testing.T) {
		queue := NewPriorityQueue[int]()

		// Insert 100 items with varying priorities
		for i := 0; i < 100; i++ {
			queue.Push(i, 100-i)
		}

		// Verify items come out in reverse order (the highest to lowest priority)
		for i := 99; i >= 0; i-- {
			val, ok := queue.Pop()
			if !ok {
				t.Errorf("Pop() at index %d should return true", i)
			}
			if val != i {
				t.Errorf("expected %d, got %d", i, val)
			}
		}
	})
}
