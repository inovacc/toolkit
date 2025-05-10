package random

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// RandomString generates a random alphanumeric string of length `n`.
// It uses crypto/rand for cryptographically secure randomness.
func RandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		// Secure a random selection of an index from letters
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[num.Int64()]
	}
	return string(b)
}

// RandomInt returns a cryptographically secure random integer between `min` and `max`.
// Returns an error if min >= max.
func RandomInt(min, max int) (int, error) {
	if min >= max {
		return 0, fmt.Errorf("invalid range: min (%d) must be less than max (%d)", min, max)
	}
	diff := max - min
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	return int(num.Int64()) + min, nil
}

// RandomBytes generates a slice of `n` random bytes using crypto/rand.
// Returns an error if random byte generation fails.
func RandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}
