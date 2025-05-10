package password

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/inovacc/toolkit/data/algorithm/random"
	"github.com/inovacc/toolkit/data/serde/encoder"
	"golang.org/x/crypto/argon2"
)

// Params holds configuration for the Argon2ID hash function.
// It includes memory usage, number of iterations, parallelism,
// salt length, and resulting key length.
type Params struct {
	Memory      uint32 `json:"m"` // Memory in KB
	Iterations  uint32 `json:"i"` // Number of iterations
	Parallelism uint8  `json:"p"` // Number of parallel threads
	SaltLength  uint32 `json:"s"` // Length of the random salt
	KeyLength   uint32 `json:"k"` // Desired length of the resulting key
}

// Hash represents the encoded structure used to store a hashed password,
// including the derived hash, salt, and the parameters used.
type Hash struct {
	Data   []byte  `json:"data"`
	Salt   []byte  `json:"salt"`
	Params *Params `json:"params"`
}

var params *Params

func init() {
	// Default Argon2ID parameters optimized for balance between security and performance.
	params = &Params{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// hashPassword generates a raw Argon2ID key using the provided password, salt, and parameters.
func hashPassword(password string, salt []byte, p *Params) ([]byte, error) {
	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	return hash, nil
}

// HashPasswordArgon2 generates a secure, encoded Argon2ID hash string using Base58.
// It marshals the hash, salt, and params into a JSON structure and encodes it.
// Returns a Base58 encoded string safe to store in databases or configuration.
func HashPasswordArgon2(password string, p *Params) (string, error) {
	if p == nil {
		p = params
	}
	if p.Memory < 1 || p.Iterations < 1 || p.Parallelism < 1 || p.SaltLength < 1 || p.KeyLength < 1 {
		return "", errors.New("invalid parameters")
	}
	if len(password) == 0 {
		return "", errors.New("password cannot be empty")
	}

	salt, _ := random.RandomBytes(p.SaltLength)
	hash, err := hashPassword(password, salt, p)
	if err != nil {
		return "", err
	}

	stored := &Hash{
		Data:   hash,
		Salt:   salt,
		Params: p,
	}

	encoded, err := json.Marshal(stored)
	if err != nil {
		return "", err
	}

	enc := encoder.NewEncoding(encoder.Base58)
	dat, err := enc.Encode(encoded)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}

// CheckPasswordHashArgon2 verifies if a given plain password matches the Base58-encoded hash.
// Returns true if they match, false otherwise.
func CheckPasswordHashArgon2(encoded, password string) (bool, error) {
	var stored Hash

	enc := encoder.NewEncoding(encoder.Base58)
	decode, err := enc.DecodeStr(encoded)
	if err != nil {
		return false, errors.New("invalid encoding hash")
	}

	if len(decode) == 0 {
		return false, errors.New("invalid hash length")
	}

	if err := json.Unmarshal([]byte(decode), &stored); err != nil {
		return false, errors.New("invalid unmarshal hash")
	}

	hash, err := hashPassword(password, stored.Salt, stored.Params)
	if err != nil {
		return false, err
	}
	return bytes.Equal(hash, stored.Data), nil
}
