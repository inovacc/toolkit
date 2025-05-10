package compression

import (
	"github.com/inovacc/toolkit/compression/brotli"
	"github.com/inovacc/toolkit/compression/gzip"
	"github.com/inovacc/toolkit/compression/lz4"
	"github.com/inovacc/toolkit/compression/snappy"
	"github.com/inovacc/toolkit/compression/zip"
	"github.com/inovacc/toolkit/compression/zlib"
	"github.com/inovacc/toolkit/compression/zstd"
)

// TypeStr defines supported compression algorithm names.
type TypeStr string

const (
	TypeZstd   TypeStr = "zstd"
	TypeGzip   TypeStr = "gzip"
	TypeSnappy TypeStr = "snappy"
	TypeLz4    TypeStr = "lz4"
	TypeBrotli TypeStr = "brotli"
	TypeZlib   TypeStr = "zlib"
	TypeZip    TypeStr = "zip"
)

// Compress holds a compression type and provides methods to compress/decompress data.
type Compress struct {
	Type TypeStr
}

// NewCompress creates a new Compress instance with the specified compression type.
func NewCompress(t TypeStr) *Compress {
	return &Compress{Type: t}
}

// Compress compresses the input byte slice using the specified compression algorithm.
func (c *Compress) Compress(data []byte) ([]byte, error) {
	switch c.Type {
	case TypeZstd:
		return zstd.Compress(data)
	case TypeGzip:
		return gzip.Compress(data)
	case TypeSnappy:
		return snappy.Compress(data)
	case TypeLz4:
		return lz4.Compress(data)
	case TypeBrotli:
		return brotli.Compress(data)
	case TypeZlib:
		return zlib.Compress(data)
	case TypeZip:
		return zip.Compress(data)
	default:
		return nil, nil
	}
}

// Decompress decompresses the input byte slice using the specified compression algorithm.
func (c *Compress) Decompress(data []byte) ([]byte, error) {
	switch c.Type {
	case TypeZstd:
		return zstd.Decompress(data)
	case TypeGzip:
		return gzip.Decompress(data)
	case TypeSnappy:
		return snappy.Decompress(data)
	case TypeLz4:
		return lz4.Decompress(data)
	case TypeBrotli:
		return brotli.Decompress(data)
	case TypeZlib:
		return zlib.Decompress(data)
	case TypeZip:
		return zip.Decompress(data)
	default:
		return nil, nil
	}
}

// String returns the name of the compression type.
func (c *Compress) String() string {
	return c.TypeString()
}

// TypeString returns the string representation of the compression type.
func (c *Compress) TypeString() string {
	switch c.Type {
	case TypeZstd, TypeGzip, TypeSnappy, TypeLz4, TypeBrotli, TypeZlib, TypeZip:
		return string(c.Type)
	default:
		return ""
	}
}
