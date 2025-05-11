package split

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// hash(32) + total(8) + size(8) + time(8) + name(50)
	metadataSize = 32 + 8 + 8 + 8 + 50
)

type Split interface {
	SplitFile(file *os.File, outDir string, chunks int) error
	MergeFile(inDir string) error
	SplitData(v any, a []any, chunks int) error
	MergeData(a []any, v any) error
}

type Impl struct {
	Name     string `json:"name"`
	Filename []byte `json:"filename"`
	Time     int64  `json:"timestamp"`
	Total    uint64 `json:"total"`
	Size     int64  `json:"size"`
	NameLen  uint16 `json:"nameLen"`
}

func NewSplit() Split {
	return &Impl{}
}

func (i *Impl) SplitFile(file *os.File, outDir string, chunks int) error {
	if chunks < 2 {
		return errors.New("chunks must be greater than or equal to 2")
	}

	fileSize := i.fileSize(file)
	size := i.calculateChunks(fileSize, chunks)
	buf := make([]byte, size)
	var index uint64

	if _, err := os.Stat(outDir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
			return err
		}
	}

	hash := sha256.New()

	var firstChunk = true
	var chunkName string

	meta := metadata{
		Total: chunks,
		Name:  file.Name(),
		Time:  time.Now().Unix(),
		Size:  fileSize,
	}

	for {
		n, err := file.Read(buf)
		if n > 0 {
			chunkFile := i.fixFileName(index, outDir, file.Name())

			if firstChunk {
				firstChunk = false
				chunkFile = strings.ReplaceAll(chunkFile, "part", "tmp")
				chunkName = chunkFile
			}

			if err := i.save(chunkFile, buf[:n]); err != nil {
				return err
			}

			hash.Write(buf[:n])
			index++
		}
		if err != nil {
			if err == io.EOF {
				meta.Hash = hash.Sum(nil)
				if err := i.injectMetadata(chunkName, meta); err != nil {
					return err
				}
				break
			}
			break
		}
	}
	return nil
}

func (i *Impl) MergeFile(inDir string) error {
	chunks, err := i.checkFiles(inDir)
	if err != nil {
		return err
	}

	meta := &metadata{}

	for _, chunk := range chunks {
		if chunk.first {
			if err := i.extractMetadata(chunk, meta); err != nil {
				return err
			}
			break
		}
	}

	wFile, err := os.Create(filepath.Join(inDir, meta.Name))
	if err != nil {
		return err
	}

	hash := sha256.New()

	for _, chunk := range chunks {
		rFile, err := os.Open(chunk.name)
		if err != nil {
			return err
		}

		if chunk.first {
			if _, err := rFile.Seek(metadataSize, 0); err != nil {
				return err
			}
		}

		reader := io.TeeReader(rFile, hash)

		if _, err := io.Copy(wFile, reader); err != nil {
			return err
		}
		rFile.Close()
	}

	sum := hash.Sum(nil)

	if !bytes.Equal(sum, meta.Hash) {
		return errors.New("merge failed, file not match")
	}

	for _, chunk := range chunks {
		if err := os.Remove(chunk.name); err != nil {
			return err
		}
	}

	fmt.Println("file restored successfully")
	return nil
}

func (i *Impl) SplitData(v any, a []any, chunks int) error {
	// TODO implement me
	panic("implement me")
}

func (i *Impl) MergeData(a []any, v any) error {
	// TODO implement me
	panic("implement me")
}

type parsedChunk struct {
	first bool
	name  string
	index int
}

func (i *Impl) checkFiles(inDir string) ([]parsedChunk, error) {
	entries, err := os.ReadDir(inDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var chunks []parsedChunk

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		chunk := filepath.Join(inDir, entry.Name())
		fileIndex, err := i.getIndex(chunk)
		if err != nil {
			return nil, err
		}

		pc := parsedChunk{name: chunk, index: fileIndex}

		if fileIndex == 0 {
			pc.first = true
		}
		chunks = append(chunks, pc)
	}

	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].index < chunks[j].index
	})

	return chunks, nil
}

func (i *Impl) calculateChunks(size int64, chunks int) int64 {
	return size/int64(chunks) + 1
}

func (i *Impl) fileSize(file *os.File) int64 {
	size, _ := file.Stat()
	return size.Size()
}

func (i *Impl) save(fileName string, data []byte) error {
	return os.WriteFile(fileName, data, 0644)
}

func (i *Impl) fixFileName(index uint64, outDir, name string) string {
	name = strings.Split(filepath.Base(name), ".")[0]
	return filepath.Join(outDir, fmt.Sprintf("%s_%04d.part", name, index))
}

func (i *Impl) getIndex(chunk string) (int, error) {
	result := -1
	parts := strings.Split(chunk, "_")
	for _, part := range parts {
		segments := strings.Split(part, ".")
		if len(segments) != 2 {
			continue
		}

		v, err := strconv.Atoi(segments[0])
		if err != nil {
			continue
		}
		result = v
	}
	return result, nil
}

type metadata struct {
	Total int
	Size  int64
	Time  int64
	Crc   uint32
	Hash  []byte
	Name  string
}

func (i *Impl) injectMetadata(chunkFile string, meta metadata) error {
	rFile, err := os.Open(chunkFile)
	if err != nil {
		return err
	}

	chunkFileNew := strings.ReplaceAll(chunkFile, "tmp", "part")
	wFile, err := os.Create(chunkFileNew)
	if err != nil {
		return err
	}

	// hash(32) + total(8) + size(8) + time(8) + name(50)
	data := make([]byte, metadataSize)

	offset := 0 // 32 bytes for file size

	// TODO work around to save hash as binary
	{
		var hashValues []uint64
		for i := 0; i+8 <= 32; i += 8 {
			part := binary.BigEndian.Uint64(meta.Hash[i : i+8])
			hashValues = append(hashValues, part)
		}

		for i, v := range hashValues {
			offset = i * 8
			binary.BigEndian.PutUint64(data[offset:], v)
		}
	}

	offset += 8
	binary.BigEndian.PutUint64(data[offset:], uint64(meta.Total))

	offset += 8 // 8 bytes for file size
	binary.BigEndian.PutUint64(data[offset:], uint64(meta.Size))

	offset += 8 // 8 bytes for file time
	binary.BigEndian.PutUint64(data[offset:], uint64(meta.Time))

	offset += 8 // 50 bytes for file name
	nameBytes := []byte(meta.Name)
	copy(data[offset:], nameBytes)

	if _, err := io.Copy(wFile, bytes.NewReader(data)); err != nil {
		return err
	}

	if _, err := io.Copy(wFile, rFile); err != nil {
		return err
	}

	rFile.Close()
	wFile.Close()
	os.Remove(chunkFile)

	return nil
}

func (i *Impl) extractMetadata(chunk parsedChunk, meta *metadata) error {
	rFile, err := os.Open(chunk.name)
	if err != nil {
		return err
	}

	// hash(32) + total(8) + size(8) + time(8) + name(50)
	data := make([]byte, metadataSize)

	_, err = rFile.Read(data)
	if err != nil {
		return err
	}

	rFile.Close()

	// TODO work around to reconstruct hash from uitn64
	{
		var hashParts []uint64
		for i := 0; i+8 <= 32; i += 8 {
			v := binary.BigEndian.Uint64(data[i : i+8])
			hashParts = append(hashParts, v)
		}

		for _, v := range hashParts {
			temp := make([]byte, 8)
			binary.BigEndian.PutUint64(temp, v)
			meta.Hash = append(meta.Hash, temp...)
		}
	}

	offset := 40 // 32 bytes for file size
	meta.Total = int(binary.BigEndian.Uint64(data[offset:]))

	offset += 8 // 8 bytes for file size
	meta.Size = int64(binary.BigEndian.Uint64(data[offset:]))

	offset += 8 // 8 bytes for file time
	meta.Time = int64(binary.BigEndian.Uint64(data[offset:]))

	offset += 8 // 50 bytes for file name
	meta.Name = strings.SplitN(string(data[offset:]), "\x00", 2)[0]

	return nil
}
