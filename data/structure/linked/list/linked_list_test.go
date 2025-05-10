package ds

import (
	"testing"
)

func TestSinglyLinkedList(t *testing.T) {
	t.Run("new list should be empty", func(t *testing.T) {
		list := NewLinkedList[int]()
		if list.Len() != 0 {
			t.Errorf("expected length 0, got %d", list.Len())
		}
	})

	t.Run("insert operations", func(t *testing.T) {
		list := NewLinkedList[string]()

		// Test Insert
		list.Insert("first")
		list.Insert("second")
		list.Insert("third")

		if list.Len() != 3 {
			t.Errorf("expected length 3, got %d", list.Len())
		}

		// Verify values
		val, ok := list.Get(0)
		if !ok || val != "first" {
			t.Errorf("expected 'first', got %v", val)
		}

		val, ok = list.Get(2)
		if !ok || val != "third" {
			t.Errorf("expected 'third', got %v", val)
		}
	})

	t.Run("insert at position", func(t *testing.T) {
		list := NewLinkedList[int]()

		// Test InsertAt
		ok := list.InsertAt(1, 0) // [1]
		if !ok {
			t.Error("InsertAt(1, 0) should return true")
		}

		ok = list.InsertAt(2, 1) // [1, 2]
		if !ok {
			t.Error("InsertAt(2, 1) should return true")
		}

		ok = list.InsertAt(3, 1) // [1, 3, 2]
		if !ok {
			t.Error("InsertAt(3, 1) should return true")
		}

		// Test invalid positions
		if list.InsertAt(4, -1) {
			t.Error("InsertAt with negative position should return false")
		}

		if list.InsertAt(4, 4) {
			t.Error("InsertAt with position > length should return false")
		}

		// Verify values and order
		expected := []int{1, 3, 2}
		for i, exp := range expected {
			val, ok := list.Get(i)
			if !ok || val != exp {
				t.Errorf("at position %d: expected %d, got %d", i, exp, val)
			}
		}
	})

	t.Run("delete operations", func(t *testing.T) {
		list := NewLinkedList[int]()
		list.Insert(1)
		list.Insert(2)
		list.Insert(3)
		list.Insert(4)

		// Test DeleteAt
		ok := list.DeleteAt(1) // [1, 3, 4]
		if !ok {
			t.Error("DeleteAt(1) should return true")
		}

		if list.Len() != 3 {
			t.Errorf("expected length 3, got %d", list.Len())
		}

		// Test invalid positions
		if list.DeleteAt(-1) {
			t.Error("DeleteAt with negative position should return false")
		}

		if list.DeleteAt(3) {
			t.Error("DeleteAt with position >= length should return false")
		}

		// Verify remaining values
		expected := []int{1, 3, 4}
		for i, exp := range expected {
			val, ok := list.Get(i)
			if !ok || val != exp {
				t.Errorf("at position %d: expected %d, got %d", i, exp, val)
			}
		}
	})

	t.Run("get operations", func(t *testing.T) {
		list := NewLinkedList[int]()

		// Test Get on empty list
		_, ok := list.Get(0)
		if ok {
			t.Error("Get on empty list should return false")
		}

		list.Insert(10)
		list.Insert(20)

		// Test valid positions
		val, ok := list.Get(0)
		if !ok || val != 10 {
			t.Errorf("expected 10, got %d", val)
		}

		val, ok = list.Get(1)
		if !ok || val != 20 {
			t.Errorf("expected 20, got %d", val)
		}

		// Test invalid positions
		_, ok = list.Get(-1)
		if ok {
			t.Error("Get with negative position should return false")
		}

		_, ok = list.Get(2)
		if ok {
			t.Error("Get with position >= length should return false")
		}
	})
}
