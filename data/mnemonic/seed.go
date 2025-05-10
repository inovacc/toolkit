package mnemonic

import (
	"crypto/pbkdf2"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
)

// Seed represents a binary seed used for HD wallet generation
type Seed struct {
	Bytes []byte
}

// NewSeed create a new Seed with the given sentence and passphrase
func NewSeed(sentence string, passphrase string) *Seed {
	s := pbkdf2.Key([]byte(sentence), []byte(fmt.Sprintf("mnemonic%s", passphrase)), 2048, 64, sha512.New)
	return &Seed{s}
}

// ToHex returns the seed bytes as a hex-encoded string
func (s *Seed) ToHex() string {
	seedHex := make([]byte, hex.EncodedLen(len(s.Bytes)))
	hex.Encode(seedHex, s.Bytes)
	return string(seedHex)
}

func (s *Seed) String() string {
	return s.ToHex()
}
