package zstd

import (
	"bytes"
	"io"

	"github.com/inovacc/toolkit/compression/internal/zstd/zstd"
)

func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := zstd.NewWriter(&b)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
	r, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
