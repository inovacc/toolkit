//go:build go1.18
// +build go1.18

package s2

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/inovacc/toolkit/compression/internal/zstd/internal/fuzz"
	"github.com/inovacc/toolkit/compression/internal/zstd/internal/snapref"
)

func FuzzEncodingBlocks(f *testing.F) {
	fuzz.AddFromZip(f, "testdata/enc_regressions.zip", fuzz.TypeRaw, false)
	fuzz.AddFromZip(f, "testdata/fuzz/block-corpus-raw.zip", fuzz.TypeRaw, testing.Short())
	fuzz.AddFromZip(f, "testdata/fuzz/block-corpus-enc.zip", fuzz.TypeGoFuzz, testing.Short())

	// Fuzzing tweaks:
	const (
		// Max input size:
		maxSize = 8 << 20
	)

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxSize {
			return
		}

		writeDst := make([]byte, MaxEncodedLen(len(data)), MaxEncodedLen(len(data))+4)
		writeDst = append(writeDst, 1, 2, 3, 4)
		defer func() {
			got := writeDst[MaxEncodedLen(len(data)):]
			want := []byte{1, 2, 3, 4}
			if !bytes.Equal(got, want) {
				t.Fatalf("want %v, got %v - dest modified outside cap", want, got)
			}
		}()
		compDst := writeDst[:MaxEncodedLen(len(data)):MaxEncodedLen(len(data))] // Hard cap
		decDst := make([]byte, len(data))
		comp := Encode(compDst, data)
		decoded, err := Decode(decDst, comp)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(data, decoded) {
			t.Error("block decoder mismatch")
			return
		}
		if mel := MaxEncodedLen(len(data)); len(comp) > mel {
			t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
			return
		}
		comp = EncodeBetter(compDst, data)
		decoded, err = Decode(decDst, comp)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(data, decoded) {
			t.Error("block decoder mismatch")
			return
		}
		if mel := MaxEncodedLen(len(data)); len(comp) > mel {
			t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
			return
		}

		comp = EncodeBest(compDst, data)
		decoded, err = Decode(decDst, comp)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(data, decoded) {
			t.Error("block decoder mismatch")
			return
		}
		if mel := MaxEncodedLen(len(data)); len(comp) > mel {
			t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
			return
		}

		comp = EncodeSnappy(compDst, data)
		decoded, err = snapref.Decode(decDst, comp)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(data, decoded) {
			t.Error("block decoder mismatch")
			return
		}
		if mel := MaxEncodedLen(len(data)); len(comp) > mel {
			t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
			return
		}
		comp = EncodeSnappyBetter(compDst, data)
		decoded, err = snapref.Decode(decDst, comp)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(data, decoded) {
			t.Error("block decoder mismatch")
			return
		}
		if mel := MaxEncodedLen(len(data)); len(comp) > mel {
			t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
			return
		}

		comp = EncodeSnappyBest(compDst, data)
		decoded, err = snapref.Decode(decDst, comp)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(data, decoded) {
			t.Error("block decoder mismatch")
			return
		}
		if mel := MaxEncodedLen(len(data)); len(comp) > mel {
			t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
			return
		}

		concat, err := ConcatBlocks(nil, data, []byte{0})
		if err != nil || concat == nil {
			return
		}

		EstimateBlockSize(data)
		encoded := make([]byte, MaxEncodedLen(len(data)))
		if len(encoded) < MaxEncodedLen(len(data)) || minNonLiteralBlockSize > len(data) || len(data) > maxBlockSize {
			return
		}

		encodeBlockGo(encoded, data)
		encodeBlockBetterGo(encoded, data)
		encodeBlockSnappyGo(encoded, data)
		encodeBlockBetterSnappyGo(encoded, data)
		if len(data) <= 64<<10 {
			encodeBlockGo64K(encoded, data)
			encodeBlockSnappyGo64K(encoded, data)
			encodeBlockBetterGo64K(encoded, data)
			encodeBlockBetterSnappyGo64K(encoded, data)
		}
		dst := encodeGo(encoded, data)
		if dst == nil {
			return
		}
	})
}

func FuzzStreamDecode(f *testing.F) {
	enc := NewWriter(nil, WriterBlockSize(8<<10))
	addCompressed := func(b []byte) {
		var buf bytes.Buffer
		enc.Reset(&buf)
		enc.Write(b)
		enc.Close()
		f.Add(buf.Bytes())
	}
	fuzz.ReturnFromZip(f, "testdata/enc_regressions.zip", fuzz.TypeRaw, addCompressed)
	fuzz.ReturnFromZip(f, "testdata/fuzz/block-corpus-raw.zip", fuzz.TypeRaw, addCompressed)
	fuzz.ReturnFromZip(f, "testdata/fuzz/block-corpus-enc.zip", fuzz.TypeGoFuzz, addCompressed)
	dec := NewReader(nil, ReaderIgnoreCRC())
	f.Fuzz(func(t *testing.T, data []byte) {
		// Using Read
		dec.Reset(bytes.NewReader(data))
		io.Copy(io.Discard, dec)

		// Using DecodeConcurrent
		dec.Reset(bytes.NewReader(data))
		dec.DecodeConcurrent(io.Discard, 2)

		// Use ByteReader.
		dec.Reset(bytes.NewReader(data))
		for {
			_, err := dec.ReadByte()
			if err != nil {
				break
			}
		}
	})
}

func FuzzDecodeBlock(f *testing.F) {
	addCompressed := func(b []byte) {
		b2 := Encode(nil, b)
		f.Add(b2)
		f.Add(EncodeBetter(nil, b))
	}
	fuzz.ReturnFromZip(f, "testdata/enc_regressions.zip", fuzz.TypeRaw, addCompressed)
	fuzz.ReturnFromZip(f, "testdata/fuzz/block-corpus-raw.zip", fuzz.TypeRaw, addCompressed)
	fuzz.ReturnFromZip(f, "testdata/fuzz/block-corpus-enc.zip", fuzz.TypeGoFuzz, addCompressed)
	fuzz.AddFromZip(f, "testdata/dec-block-regressions.zip", fuzz.TypeRaw, false)

	f.Fuzz(func(t *testing.T, data []byte) {
		if t.Failed() {
			return
		}

		dCopy := append([]byte{}, data...)
		dlen, err := DecodedLen(data)
		if dlen > 8<<20 {
			return
		}
		base, baseErr := Decode(nil, data)
		if !bytes.Equal(data, dCopy) {
			t.Fatal("data was changed")
		}
		hasErr := baseErr != nil
		dataCapped := make([]byte, 0, len(data)+1024)
		dataCapped = append(dataCapped, data...)
		dataCapped = append(dataCapped, bytes.Repeat([]byte{0xff, 0xff, 0xff, 0xff}, 1024/4)...)
		dataCapped = dataCapped[:len(data):len(data)]
		if dlen > MaxBlockSize {
			dlen = MaxBlockSize
		}
		dst2 := bytes.Repeat([]byte{0xfe}, dlen+1024)
		got, err := Decode(dst2[:dlen:dlen], dataCapped[:len(data)])
		if !bytes.Equal(dataCapped[:len(data)], dCopy) {
			t.Fatal("data was changed")
		}
		if err != nil && !hasErr {
			t.Fatalf("base err: %v, capped: %v", baseErr, err)
		}
		for i, v := range dst2[dlen:] {
			if v != 0xfe {
				t.Errorf("DST overwritten beyond cap! index %d: got 0x%02x, want 0x%02x, err:%v", i, v, 0xfe, err)
				break
			}
		}
		if baseErr == nil {
			if !bytes.Equal(got, base) {
				t.Fatal("data mismatch")
			}
			gotLen, err := DecodedLen(data)
			if err != nil {
				t.Errorf("DecodedLen returned error: %v", err)
			} else if gotLen != len(got) {
				t.Errorf("DecodedLen mismatch: got %d, want %d", gotLen, len(got))
			}
		}
	})
}
