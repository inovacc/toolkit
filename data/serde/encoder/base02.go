package encoder

import (
	"bytes"
	"errors"
	"fmt"
)

type base02Encoding struct {
	limit int
}

func newBase02Encoding() *base02Encoding {
	return &base02Encoding{}
}

func (b *base02Encoding) SetLimit(limit int) {
	b.limit = limit
}

func (b *base02Encoding) EncodeStr(s string) (string, error) {
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

func (b *base02Encoding) DecodeStr(s string) (string, error) {
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

func (b *base02Encoding) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	for _, b := range data {
		buf.WriteString(fmt.Sprintf("%08b", b))
	}
	return buf.Bytes(), nil
}

func (b *base02Encoding) Decode(data []byte) ([]byte, error) {
	str := string(data)
	if len(str)%8 != 0 {
		return nil, errors.New("binary data length must be a multiple of 8")
	}

	out := make([]byte, len(str)/8)
	for i := 0; i < len(out); i++ {
		var b byte
		slice := str[i*8 : (i+1)*8]
		if _, err := fmt.Sscanf(slice, "%08b", &b); err != nil {
			return nil, fmt.Errorf("invalid binary segment '%s': %w", slice, err)
		}
		out[i] = b
	}
	return out, nil
}
