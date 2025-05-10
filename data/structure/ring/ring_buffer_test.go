package ds

import "testing"

func TestRingBuffer(t *testing.T) {
	t.Run("new buffer properties", func(t *testing.T) {
		rb, _ := NewRingBuffer[int](5)
		if rb.Len() != 0 {
			t.Errorf("expected length 0, got %d", rb.Len())
		}
		if rb.Cap() != 5 {
			t.Errorf("expected capacity 5, got %d", rb.Cap())
		}

		// Check empty buffer operations
		_, ok := rb.Pop()
		if ok {
			t.Error("Pop on empty buffer should return false")
		}

		_, ok = rb.Peek()
		if ok {
			t.Error("Peek on empty buffer should return false")
		}
	})

	t.Run("basic operations", func(t *testing.T) {
		rb, _ := NewRingBuffer[string](3)

		// Test Push operations
		rb.Push("first")
		if rb.Len() != 1 {
			t.Errorf("expected length 1, got %d", rb.Len())
		}

		// Test Peek
		val, ok := rb.Peek()
		if !ok || val != "first" {
			t.Errorf("expected Peek to return 'first', got %v", val)
		}

		// Test Pop
		val, ok = rb.Pop()
		if !ok || val != "first" {
			t.Errorf("expected Pop to return 'first', got %v", val)
		}
		if rb.Len() != 0 {
			t.Errorf("expected length 0 after Pop, got %d", rb.Len())
		}
	})

	t.Run("buffer overflow", func(t *testing.T) {
		rb, _ := NewRingBuffer[int](3)

		// Fill buffer
		for i := 0; i < 3; i++ {
			rb.Push(i)
		}
		if rb.Len() != 3 {
			t.Errorf("expected length 3, got %d", rb.Len())
		}

		// Add one more element, should override the oldest
		rb.Push(3)
		if rb.Len() != 3 {
			t.Errorf("expected length to remain 3, got %d", rb.Len())
		}

		// Check values - should be [1,2,3], not [0,1,2]
		expected := []int{1, 2, 3}
		for i, exp := range expected {
			val, ok := rb.Pop()
			if !ok {
				t.Errorf("Pop at index %d should return true", i)
			}
			if val != exp {
				t.Errorf("expected %d, got %d", exp, val)
			}
		}
	})

	t.Run("wrap around", func(t *testing.T) {
		rb, _ := NewRingBuffer[int](3)

		// Fill and partially empty
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)
		rb.Pop() // remove 1
		rb.Pop() // remove 2

		// Add new elements
		rb.Push(4)
		rb.Push(5)

		// Check values - should be [3,4,5]
		expected := []int{3, 4, 5}
		for i, exp := range expected {
			val, ok := rb.Pop()
			if !ok {
				t.Errorf("Pop at index %d should return true", i)
			}
			if val != exp {
				t.Errorf("expected %d, got %d", exp, val)
			}
		}
	})

	t.Run("alternating operations", func(t *testing.T) {
		rb, _ := NewRingBuffer[int](4)

		operations := []struct {
			push    bool
			value   int
			expVal  int
			expOk   bool
			expLen  int
			comment string
		}{
			{true, 1, 0, false, 1, "push 1"},
			{true, 2, 0, false, 2, "push 2"},
			{false, 0, 1, true, 1, "pop 1"},
			{true, 3, 0, false, 2, "push 3"},
			{true, 4, 0, false, 3, "push 4"},
			{false, 0, 2, true, 2, "pop 2"},
			{false, 0, 3, true, 1, "pop 3"},
			{true, 5, 0, false, 2, "push 5"},
			{false, 0, 4, true, 1, "pop 4"},
			{false, 0, 5, true, 0, "pop 5"},
		}

		for i, op := range operations {
			if op.push {
				rb.Push(op.value)
			} else {
				val, ok := rb.Pop()
				if ok != op.expOk || val != op.expVal {
					t.Errorf("step %d (%s): expected (%d, %v), got (%d, %v)",
						i, op.comment, op.expVal, op.expOk, val, ok)
				}
			}
			if rb.Len() != op.expLen {
				t.Errorf("step %d (%s): expected length %d, got %d",
					i, op.comment, op.expLen, rb.Len())
			}
		}
	})

	t.Run("invalid capacity", func(t *testing.T) {
		rb, err := NewRingBuffer[int](0)
		if err == nil {
			t.Error("expected error for zero capacity, got nil")
		}
		if rb != nil {
			t.Error("expected nil buffer for zero capacity")
		}
	})
}
