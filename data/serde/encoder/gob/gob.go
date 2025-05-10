package gob

import (
	"bytes"
	"encoding/gob"
)

// EncodeGob serializes the given data into a byte slice using the Go gob encoding.
// The data can be any Go value, but the underlying type must be registered or gob-encodable.
// Returns the serialized byte slice or an error if encoding fails.
func EncodeGob(data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return nil, err // Encoding failed
	}
	return buf.Bytes(), nil
}

// DecodeGob deserializes a gob-encoded byte slice into the provided target variable `v`.
// The target `v` must be a pointer to the expected type.
// Returns an error if decoding fails or types mismatch.
func DecodeGob(data []byte, v any) error {
	buf := bytes.NewBuffer(data)
	if err := gob.NewDecoder(buf).Decode(v); err != nil {
		return err // Decoding failed
	}
	return nil
}
