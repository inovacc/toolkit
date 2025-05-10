package hashing

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
)

// HashFunction represents a function that creates a new hash.Hash
type HashFunction func() hash.Hash

// Common hash functions
var (
	SHA224     HashFunction = sha256.New224
	SHA256     HashFunction = sha256.New
	SHA384     HashFunction = sha512.New384
	SHA512     HashFunction = sha512.New
	SHA512_224 HashFunction = sha512.New512_224
	SHA512_256 HashFunction = sha512.New512_256
)

// Hasher provides a unified interface for hashing operations
type Hasher struct {
	newHash HashFunction
}

// NewHasher creates a new Hasher with the specified hash function
func NewHasher(hf HashFunction) *Hasher {
	return &Hasher{
		newHash: hf,
	}
}

// HashString returns the hexadecimal hash of the given string input
func (h *Hasher) HashString(data string) string {
	return h.HashBytes([]byte(data))
}

// HashBytes returns the hexadecimal hash of the given byte slice
func (h *Hasher) HashBytes(data []byte) string {
	nh := h.newHash()
	nh.Write(data)
	return hex.EncodeToString(nh.Sum(nil))
}

// GetSize returns the size of the hash output in bytes
func (h *Hasher) GetSize() int {
	return h.newHash().Size()
}

// GetBlockSize returns the block size of the hash in bytes
func (h *Hasher) GetBlockSize() int {
	return h.newHash().BlockSize()
}

// Reset creates a new hash.Hash instance
func (h *Hasher) Reset() hash.Hash {
	return h.newHash()
}

// CompareString compares two hashes and returns true if they are equal
func (h *Hasher) CompareString(a, b string) bool {
	return h.CompareBytes([]byte(a), []byte(b))
}

// CompareBytes compares two hashes and returns true if they are equal
func (h *Hasher) CompareBytes(a, b []byte) bool {
	return bytes.Equal(a, b)
}
