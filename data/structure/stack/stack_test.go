package ds

import "testing"

func TestStack(t *testing.T) {
	t.Run("new stack properties", func(t *testing.T) {
		stack := NewStack[int]()
		if !stack.IsEmpty() {
			t.Error("new stack should be empty")
		}
		if stack.Len() != 0 {
			t.Errorf("expected length 0, got %d", stack.Len())
		}

		// Check empty stack operations
		_, ok := stack.Pop()
		if ok {
			t.Error("Pop on empty stack should return false")
		}

		_, ok = stack.Peek()
		if ok {
			t.Error("Peek on empty stack should return false")
		}
	})

	t.Run("basic operations", func(t *testing.T) {
		stack := NewStack[string]()

		// Test Push
		stack.Push("first")
		if stack.IsEmpty() {
			t.Error("stack should not be empty after Push")
		}
		if stack.Len() != 1 {
			t.Errorf("expected length 1, got %d", stack.Len())
		}

		// Test Peek
		val, ok := stack.Peek()
		if !ok {
			t.Error("Peek should return true for non-empty stack")
		}
		if val != "first" {
			t.Errorf("expected Peek to return 'first', got %v", val)
		}
		if stack.Len() != 1 {
			t.Error("Peek should not modify stack length")
		}

		// Test Pop
		val, ok = stack.Pop()
		if !ok {
			t.Error("Pop should return true for non-empty stack")
		}
		if val != "first" {
			t.Errorf("expected Pop to return 'first', got %v", val)
		}
		if !stack.IsEmpty() {
			t.Error("stack should be empty after Pop")
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		stack := NewStack[int]()
		numbers := []int{1, 2, 3, 4, 5}

		// Push all numbers
		for _, n := range numbers {
			stack.Push(n)
		}

		if stack.Len() != len(numbers) {
			t.Errorf("expected length %d, got %d", len(numbers), stack.Len())
		}

		// Pop all numbers and verify LIFO order
		for i := len(numbers) - 1; i >= 0; i-- {
			val, ok := stack.Pop()
			if !ok {
				t.Errorf("Pop failed at index %d", i)
			}
			if val != numbers[i] {
				t.Errorf("expected %d, got %d", numbers[i], val)
			}
		}

		if !stack.IsEmpty() {
			t.Error("stack should be empty after popping all elements")
		}
	})

	t.Run("mixed types", func(t *testing.T) {
		t.Run("floats", func(t *testing.T) {
			stack := NewStack[float64]()
			stack.Push(1.1)
			stack.Push(2.2)
			val, ok := stack.Pop()
			if !ok || val != 2.2 {
				t.Errorf("expected 2.2, got %v", val)
			}
		})

		t.Run("structs", func(t *testing.T) {
			type TestStruct struct {
				value int
			}
			stack := NewStack[TestStruct]()
			stack.Push(TestStruct{1})
			stack.Push(TestStruct{2})
			val, ok := stack.Pop()
			if !ok || val.value != 2 {
				t.Errorf("expected struct with value 2, got %v", val.value)
			}
		})
	})

	t.Run("alternating operations", func(t *testing.T) {
		stack := NewStack[int]()
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
			{false, 0, 2, true, 1, "pop 2"},
			{true, 3, 0, false, 2, "push 3"},
			{false, 0, 3, true, 1, "pop 3"},
			{false, 0, 1, true, 0, "pop 1"},
			{false, 0, 0, false, 0, "pop empty"},
		}

		for i, op := range operations {
			if op.push {
				stack.Push(op.value)
			} else {
				val, ok := stack.Pop()
				if ok != op.expOk || val != op.expVal {
					t.Errorf("step %d (%s): expected (%d, %v), got (%d, %v)",
						i, op.comment, op.expVal, op.expOk, val, ok)
				}
			}
			if stack.Len() != op.expLen {
				t.Errorf("step %d (%s): expected length %d, got %d",
					i, op.comment, op.expLen, stack.Len())
			}
		}
	})

	t.Run("zero values", func(t *testing.T) {
		stack := NewStack[int]()
		stack.Push(0) // Push zero value

		val, ok := stack.Peek()
		if !ok {
			t.Error("Peek should succeed for zero value")
		}
		if val != 0 {
			t.Error("Peek should return pushed zero value")
		}

		val, ok = stack.Pop()
		if !ok {
			t.Error("Pop should succeed for zero value")
		}
		if val != 0 {
			t.Error("Pop should return pushed zero value")
		}
	})
}
