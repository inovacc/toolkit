package ds

// Set is a generic interface for a collection of unique elements.
type Set[T comparable] interface {
	// Add inserts a value into the set.
	Add(value T)

	// Remove deletes a value from the set.
	Remove(value T)

	// Has checks if a value exists in the set.
	Has(value T) bool

	// Len returns the number of elements in the set.
	Len() int

	// Items return a slice containing all elements in the set.
	Items() []T
}

type HashSet[T comparable] struct {
	data map[T]struct{}
}

func NewSet[T comparable]() *HashSet[T] {
	return &HashSet[T]{data: make(map[T]struct{})}
}

func (s *HashSet[T]) Add(value T) {
	s.data[value] = struct{}{}
}

func (s *HashSet[T]) Remove(value T) {
	delete(s.data, value)
}

func (s *HashSet[T]) Has(value T) bool {
	_, exists := s.data[value]
	return exists
}

func (s *HashSet[T]) Len() int {
	return len(s.data)
}

func (s *HashSet[T]) Items() []T {
	values := make([]T, 0, len(s.data))
	for k := range s.data {
		values = append(values, k)
	}
	return values
}
