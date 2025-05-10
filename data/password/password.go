package password

import (
	"crypto/rand"
	"errors"
	"math/big"
)

// Option is a functional option type for configuring password generation.
type Option func(*Password)

// passwordOptions defines character sets for password generation.
var passwordOptions = map[string]string{
	"num":         "1234567890",
	"specialChar": "!@#$%&'()*+,^-./:;<=>?[]_`{~}|",
	"lowerCase":   "abcdefghijklmnopqrstuvwxyz",
	"upperCase":   "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
}

// Password holds configuration for generating passwords.
type Password struct {
	length      int
	hasNumbers  bool
	hashSpecial bool
	hasLower    bool
	hasUpper    bool
}

// NewPassword constructs a Password with provided options.
// By default, it creates a password of length 8 with no character types enabled.
func NewPassword(opts ...Option) *Password {
	p := &Password{
		length: 8,
	}

	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithLength sets the length of the password (8–128).
func WithLength(length int) Option {
	return func(p *Password) {
		p.length = length
	}
}

// WithNumbers enables or disables numeric characters in the password.
func WithNumbers() Option {
	return func(p *Password) {
		p.hasNumbers = true
	}
}

// WithSpecial enables or disables special characters in the password.
func WithSpecial() Option {
	return func(p *Password) {
		p.hashSpecial = true
	}
}

// WithLower enables or disables lowercase letters in the password.
func WithLower() Option {
	return func(p *Password) {
		p.hasLower = true
	}
}

// WithUpper enables or disables uppercase letters in the password.
func WithUpper() Option {
	return func(p *Password) {
		p.hasUpper = true
	}
}

// Generate builds a password using the configured options.
// It ensures at least one character from each enabled category is included.
func (p *Password) Generate() (string, error) {
	if p.length < 8 || p.length > 128 {
		return "", errors.New("length must be between 8 and 128 characters")
	}

	// Create a slice of enabled character sets
	var enabledSets []string
	if p.hasNumbers {
		enabledSets = append(enabledSets, "num")
	}
	if p.hashSpecial {
		enabledSets = append(enabledSets, "specialChar")
	}
	if p.hasLower {
		enabledSets = append(enabledSets, "lowerCase")
	}
	if p.hasUpper {
		enabledSets = append(enabledSets, "upperCase")
	}

	if len(enabledSets) == 0 {
		return "", errors.New("no character sets selected, please enable at least one")
	}

	var passChars []rune
	charsPerSet := p.length / len(enabledSets)
	extraChars := p.length % len(enabledSets)

	// First, ensure minimum distribution from each set
	for _, setName := range enabledSets {
		currentCount := charsPerSet
		if extraChars > 0 {
			currentCount++
			extraChars--
		}

		for i := 0; i < currentCount; i++ {
			char := p.getRandomChar(passwordOptions[setName])
			passChars = append(passChars, char)
		}
	}

	// If we still need more characters (due to rounding), add them from a random enabled set
	for len(passChars) < p.length {
		randomSetIndex := p.getRandomIndex(len(enabledSets))
		setName := enabledSets[randomSetIndex]
		char := p.getRandomChar(passwordOptions[setName])
		passChars = append(passChars, char)
	}

	p.shuffleRunes(passChars)
	return string(passChars), nil
}

// getRandomChar selects one rune at random from the input string.
func (p *Password) getRandomChar(fromString string) rune {
	index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(fromString))))
	return rune(fromString[index.Int64()])
}

// shuffleRunes performs Fisher–Yates shuffle to randomize rune order.
func (p *Password) shuffleRunes(runes []rune) {
	for i := len(runes) - 1; i > 0; i-- {
		jBig, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(jBig.Int64())
		runes[i], runes[j] = runes[j], runes[i]
	}
}

// getRandomIndex returns a random index within the given range
func (p *Password) getRandomIndex(max int) int {
	index, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(index.Int64())
}
