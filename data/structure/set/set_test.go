package ds

import (
	"sort"
	"testing"
)

func TestHashSet(t *testing.T) {
	t.Run("new set properties", func(t *testing.T) {
		set := NewSet[int]()
		if set.Len() != 0 {
			t.Errorf("expected length 0, got %d", set.Len())
		}
		if len(set.Items()) != 0 {
			t.Error("expected empty items slice")
		}
	})

	t.Run("basic operations", func(t *testing.T) {
		set := NewSet[string]()

		// Test Add and Has
		set.Add("first")
		if !set.Has("first") {
			t.Error("expected set to have 'first'")
		}
		if set.Len() != 1 {
			t.Errorf("expected length 1, got %d", set.Len())
		}

		// Test duplicate add
		set.Add("first")
		if set.Len() != 1 {
			t.Error("adding duplicate should not increase length")
		}

		// Test Remove
		set.Remove("first")
		if set.Has("first") {
			t.Error("expected set to not have 'first' after removal")
		}
		if set.Len() != 0 {
			t.Errorf("expected length 0 after removal, got %d", set.Len())
		}

		// Test removing non-existent item
		set.Remove("nonexistent")
		if set.Len() != 0 {
			t.Error("removing non-existent item should not affect length")
		}
	})

	t.Run("multiple items", func(t *testing.T) {
		set := NewSet[int]()
		numbers := []int{1, 2, 3, 4, 5}

		// Add all numbers
		for _, n := range numbers {
			set.Add(n)
		}

		if set.Len() != len(numbers) {
			t.Errorf("expected length %d, got %d", len(numbers), set.Len())
		}

		// Check all numbers exist
		for _, n := range numbers {
			if !set.Has(n) {
				t.Errorf("expected set to have %d", n)
			}
		}

		// Verify Items returns all values
		items := set.Items()
		if len(items) != len(numbers) {
			t.Errorf("expected Items() to return %d elements, got %d", len(numbers), len(items))
		}

		// Sort both slices for comparison
		sort.Ints(items)
		sort.Ints(numbers)
		for i := range numbers {
			if items[i] != numbers[i] {
				t.Errorf("Items() mismatch at index %d: expected %d, got %d", i, numbers[i], items[i])
			}
		}
	})

	t.Run("mixed types", func(t *testing.T) {
		t.Run("strings", func(t *testing.T) {
			set := NewSet[string]()
			words := []string{"apple", "banana", "cherry"}
			for _, w := range words {
				set.Add(w)
			}
			if set.Len() != len(words) {
				t.Errorf("expected length %d, got %d", len(words), set.Len())
			}
		})

		t.Run("floats", func(t *testing.T) {
			set := NewSet[float64]()
			nums := []float64{1.1, 2.2, 3.3}
			for _, n := range nums {
				set.Add(n)
			}
			if set.Len() != len(nums) {
				t.Errorf("expected length %d, got %d", len(nums), set.Len())
			}
		})
	})

	t.Run("remove operations", func(t *testing.T) {
		set := NewSet[int]()
		nums := []int{1, 2, 3, 4, 5}

		// Add all numbers
		for _, n := range nums {
			set.Add(n)
		}

		// Remove even numbers
		set.Remove(2)
		set.Remove(4)

		// Check remaining numbers
		expected := []int{1, 3, 5}
		items := set.Items()
		sort.Ints(items)

		if set.Len() != len(expected) {
			t.Errorf("expected length %d, got %d", len(expected), set.Len())
		}

		for i, n := range expected {
			if items[i] != n {
				t.Errorf("expected %d at index %d, got %d", n, i, items[i])
			}
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		set := NewSet[string]()

		// Empty string
		set.Add("")
		if !set.Has("") {
			t.Error("set should be able to store empty string")
		}

		// Remove empty string
		set.Remove("")
		if set.Has("") {
			t.Error("empty string should be removed")
		}

		// Zero value operations
		var zero string
		set.Add(zero)
		if !set.Has(zero) {
			t.Error("set should be able to store zero value")
		}
	})
}
