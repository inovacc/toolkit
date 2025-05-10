package encoder

import (
	"encoding/base64"
)

type base64Encoding struct {
	limit int
}

func newBase64Encoding() *base64Encoding {
	return &base64Encoding{}
}

func (b *base64Encoding) SetLimit(limit int) {
	b.limit = limit
}

func (b *base64Encoding) EncodeStr(s string) (string, error) {
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

func (b *base64Encoding) DecodeStr(s string) (string, error) {
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

func (b *base64Encoding) Encode(data []byte) ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(data)), nil
}

func (b *base64Encoding) Decode(data []byte) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(data))
}
