package entropy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Bits represent a byte slice of individual bits
type Bits []byte

// FromHex creates entropy bits from a hex string
func FromHex(input string) ([]byte, error) {
	bytes, err := hex.DecodeString(input)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// Random creates random entropy of the given length
func Random(length int) ([]byte, error) {
	if length < 128 || length > 256 || length%32 > 0 {
		return nil, errors.New("entropy length must be between 128 and 256 inclusive, and be divisible by 32")
	}
	bytes := make([]byte, length/8)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// CheckSum returns a slice of bits from the given entropy
func CheckSum(ent []byte) Bits {
	h := sha256.New()
	h.Write(ent)
	cs := h.Sum(nil)
	hashBits := bytesToBits(cs)
	num := len(ent) * 8 / 32
	return hashBits[:num]
}

// CheckSummed returns slice of entropy a bit with an appended check sum
func CheckSummed(ent []byte) Bits {
	cs := CheckSum(ent)
	bits := bytesToBits(ent)
	return append(bits, cs...)
}

func bytesToBits(bytes []byte) Bits {
	length := len(bytes)
	bits := make([]byte, length*8)
	for i := 0; i < length; i++ {
		b := bytes[i]
		for j := 0; j < 8; j++ {
			mask := byte(1 << uint8(j))
			bit := b & mask
			if bit == 0 {
				bits[(i*8)+8-(j+1)] = '0'
			} else {
				bits[(i*8)+8-(j+1)] = '1'
			}
		}
	}
	return bits
}
