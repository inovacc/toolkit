package mnemonic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/inovacc/toolkit/data/mnemonic/entropy"
)

// Mnemonic represents a collection of human-readable words
// used for HD wallet seed generation.
type Mnemonic struct {
	Words    []string
	Language LanguageStr
}

// New creates a new Mnemonic from given entropy and language.
func New(ent []byte, lang LanguageStr) (*Mnemonic, error) {
	bitLength := len(ent) * 8
	if bitLength < 128 || bitLength > 256 || bitLength%32 != 0 {
		return nil, fmt.Errorf("invalid entropy length: %d bits (must be 128–256 and divisible by 32)", bitLength)
	}

	const chunkSize = 11

	bits := entropy.CheckSummed(ent)
	if len(bits)%chunkSize != 0 {
		return nil, fmt.Errorf("invalid checksummed entropy length: must be divisible by %d", chunkSize)
	}

	wordCount := len(bits) / chunkSize
	words := make([]string, wordCount)

	for i := 0; i < wordCount; i++ {
		start := i * chunkSize
		end := start + chunkSize
		bitChunk := bits[start:end]

		index, err := strconv.ParseInt(string(bitChunk), 2, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid binary chunk '%s' at index %d: %w", bitChunk, i, err)
		}

		words[i] = GetWord(lang, int(index))
	}

	return &Mnemonic{
		Words:    words,
		Language: lang,
	}, nil
}

// NewRandom generates a Mnemonic using random entropy of a given bit length.
func NewRandom(bitLength int, lang LanguageStr) (*Mnemonic, error) {
	ent, err := entropy.Random(bitLength)
	if err != nil {
		return nil, fmt.Errorf("error generating random entropy: %w", err)
	}
	return New(ent, lang)
}

// Sentence returns the Mnemonic words joined as a phrase.
// Uses full-width space for Japanese.
func (m *Mnemonic) Sentence() string {
	separator := " "
	if m.Language == Japanese {
		separator = "　"
	}
	return strings.Join(m.Words, separator)
}

// GenerateSeed generates a seed (e.g., for BIP-0032 wallets) from the mnemonic sentence and a passphrase.
func (m *Mnemonic) GenerateSeed(passphrase string) *Seed {
	return NewSeed(m.Sentence(), passphrase)
}
