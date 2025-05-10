package zip

import (
	"archive/zip"
	"bytes"
	"io"
)

func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	f, err := w.Create("data")
	if err != nil {
		return nil, err
	}
	_, err = f.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if len(r.File) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	rc, err := r.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer func(rc io.ReadCloser) {
		_ = rc.Close()
	}(rc)
	return io.ReadAll(rc)
}
