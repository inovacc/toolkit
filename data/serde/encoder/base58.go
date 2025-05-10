package encoder

import (
	"fmt"
	"math/big"
)

const (
	alphabetString = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
)

type Alphabet struct {
	chars     string
	decodeMap [256]byte
}

func newAlphabet(chars string) *Alphabet {
	a := &Alphabet{chars: chars}
	for i := range a.decodeMap {
		a.decodeMap[i] = 0xFF
	}
	for i, c := range []byte(chars) {
		a.decodeMap[c] = byte(i)
	}
	return a
}

type base58Encoding struct {
	alphabet *Alphabet
	limit    int
}

func newBase58Encoding() *base58Encoding {
	return &base58Encoding{alphabet: newAlphabet(alphabetString)}
}

func (b *base58Encoding) SetLimit(limit int) {
	b.limit = limit
}

func (b *base58Encoding) EncodeStr(s string) (string, error) {
	data, err := b.Encode([]byte(s))
	if err != nil {
		return "", err
	}

	encoded := string(data)
	if b.limit > 0 {
		encoded = wrapString(encoded, b.limit)
	}

	return encoded, err
}

func (b *base58Encoding) DecodeStr(s string) (string, error) {
	decoded := s
	if b.limit > 0 {
		decoded = unwrapString(decoded)
	}

	data, err := b.Decode([]byte(decoded))
	if err != nil {
		return "", err
	}
	return string(data), err
}

func (b *base58Encoding) Encode(data []byte) ([]byte, error) {
	num := new(big.Int).SetBytes(data)
	mod := new(big.Int)
	output := make([]byte, 0)

	for num.Sign() > 0 {
		num.DivMod(num, big.NewInt(58), mod)
		output = append(output, b.alphabet.chars[mod.Int64()])
	}

	for i := 0; i < len(data) && data[i] == 0; i++ {
		output = append(output, b.alphabet.chars[0])
	}

	return []byte(reverse(output)), nil
}

func (b *base58Encoding) Decode(data []byte) ([]byte, error) {
	result := big.NewInt(0)
	for _, c := range data {
		val := b.alphabet.decodeMap[c]
		if val == 0xFF {
			return nil, fmt.Errorf("invalid character: %q", c)
		}
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(val)))
	}

	decoded := result.Bytes()
	zeroCount := 0
	for zeroCount < len(data) && data[zeroCount] == b.alphabet.chars[0] {
		zeroCount++
	}

	return append(make([]byte, zeroCount), decoded...), nil
}

func reverse(b []byte) string {
	for i := 0; i < len(b)/2; i++ {
		b[i], b[len(b)-1-i] = b[len(b)-1-i], b[i]
	}
	return string(b)
}
