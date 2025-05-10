package encoder

import (
	"bytes"
	"fmt"
	"math/big"
)

var (
	base62Alphabet = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
)

type base62Encoding struct {
	limit int
}

func newBase62Encoding() *base62Encoding {
	return &base62Encoding{}
}

func (b *base62Encoding) SetLimit(limit int) {
	b.limit = limit
}

func (b *base62Encoding) EncodeStr(s string) (string, error) {
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

func (b *base62Encoding) DecodeStr(s string) (string, error) {
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

func (b *base62Encoding) Encode(data []byte) ([]byte, error) {
	num := new(big.Int).SetBytes(data)
	base := big.NewInt(62)
	zero := big.NewInt(0)

	var encoded []byte
	for num.Cmp(zero) > 0 {
		mod := new(big.Int)
		num.DivMod(num, base, mod)
		encoded = append([]byte{base62Alphabet[mod.Int64()]}, encoded...)
	}
	return encoded, nil
}

func (b *base62Encoding) Decode(input []byte) ([]byte, error) {
	base := big.NewInt(62)
	result := big.NewInt(0)

	for _, c := range input {
		index := bytes.IndexByte(base62Alphabet, c)
		if index < 0 {
			return nil, fmt.Errorf("invalid character: %c", c)
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(index)))
	}

	return result.Bytes(), nil
}
