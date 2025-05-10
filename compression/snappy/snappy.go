package snappy

import "github.com/inovacc/toolkit/compression/internal/snappy"

func Compress(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

func Decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}
