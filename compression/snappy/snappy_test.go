package snappy

import (
	"bytes"
	"testing"

	"github.com/inovacc/toolkit/data/serde/encoder"
)

func TestCompress(t *testing.T) {
	data, err := Compress([]byte("test"))
	if err != nil {
		t.Errorf("Compress failed: %v", err)
		return
	}

	decompressed, err := Decompress(data)
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
		return
	}

	if !bytes.Equal(decompressed, []byte("test")) {
		t.Errorf("Decompressed data does not match original data")
		return
	}
}

func TestDecompress(t *testing.T) {
	enc := encoder.NewEncoding(encoder.Base64)
	data, err := enc.Decode([]byte("BAx0ZXN0"))
	if err != nil {
		t.Error("unable to decode data")
		return
	}

	decompressed, err := Decompress(data)
	if err != nil {
		t.Errorf("Decompress failed: %v", err)
		return
	}

	if !bytes.Equal(decompressed, []byte("test")) {
		t.Errorf("Decompressed data does not match original data")
		return
	}
}
