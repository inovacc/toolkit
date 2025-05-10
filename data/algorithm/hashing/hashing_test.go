package hashing

import (
	"testing"
)

func TestHasher(t *testing.T) {
	tests := []struct {
		name     string
		hashFunc HashFunction
		input    string
	}{
		{
			name:     "SHA-224",
			hashFunc: SHA224,
			input:    "test",
		},
		{
			name:     "SHA-256",
			hashFunc: SHA256,
			input:    "test",
		},
		{
			name:     "SHA-384",
			hashFunc: SHA384,
			input:    "test",
		},
		{
			name:     "SHA-512",
			hashFunc: SHA512,
			input:    "test",
		},
		{
			name:     "SHA-512/224",
			hashFunc: SHA512_224,
			input:    "test",
		},
		{
			name:     "SHA-512/256",
			hashFunc: SHA512_256,
			input:    "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewHasher(tt.hashFunc)
			hash := hasher.HashString(tt.input)
			hashBytes := hasher.HashBytes([]byte(tt.input))

			if hash != hashBytes {
				t.Errorf("HashString and HashBytes produced different results:\nString: %s\nBytes: %s", hash, hashBytes)
			}
		})
	}
}
