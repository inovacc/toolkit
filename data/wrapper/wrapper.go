package wrapper

import (
	"encoding/json"
)

// Wrapper is an interface that provides JSON marshaling capabilities
type Wrapper[T any] interface {
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
	Clone(empty bool) Wrapper[T]
	Get() T
	Set(data T)
}

// WithJSON adds JSON marshaling capabilities to any struct
type WithJSON[T any] struct {
	Data T
}

// NewWrapper creates a new instance of withJSON
func NewWrapper[T any](data T) *WithJSON[T] {
	return &WithJSON[T]{
		Data: data,
	}
}

// MarshalJSON implements the json.Marshaler interface
func (w *WithJSON[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.Data)
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (w *WithJSON[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &w.Data)
}

// Get returns the underlying data
func (w *WithJSON[T]) Get() T {
	return w.Data
}

// Set updates the underlying data
func (w *WithJSON[T]) Set(data T) {
	w.Data = data
}

// Clone returns a new wrapper with a deep copy of the data
func (w *WithJSON[T]) Clone(empty bool) Wrapper[T] {
	if empty {
		var zero T
		return NewWrapper(zero)
	}

	// Marshal the original data
	data, err := json.Marshal(w.Data)
	if err != nil {
		// Since we can't return an error in the interface,
		// return an empty wrapper as fallback
		var zero T
		return NewWrapper(zero)
	}

	// Create a new instance
	var newData T

	// Unmarshal into the new instance
	if err := json.Unmarshal(data, &newData); err != nil {
		// Return empty wrapper as fallback
		var zero T
		return NewWrapper(zero)
	}

	return NewWrapper(newData)
}
