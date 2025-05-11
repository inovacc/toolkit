package split

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type metadata struct {
	Hash  [32]byte // 32 bytes SHA-256
	Total uint32   // 4 bytes
	Size  int64    // 8 bytes
	Time  int64    // 8 bytes
	Name  [46]byte // truncated or padded filename
}

type Split struct {
	Name     string `json:"name"`
	Filename []byte `json:"filename"`
	Time     int64  `json:"timestamp"`
	Total    uint64 `json:"total"`
	Size     int64  `json:"size"`
	NameLen  uint16 `json:"nameLen"`
}

func NewSplit() *Split {
	return &Split{}
}

func (s *Split) SplitFile(file *os.File, outDir string, chunks int) error {
	if chunks < 2 {
		return errors.New("chunks must be at least 2")
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()
	chunkSize := fileSize/int64(chunks) + 1
	buf := make([]byte, chunkSize)

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	hash := sha256.New()
	nameBase := filepath.Base(file.Name())
	meta := metadata{
		Total: uint32(chunks),
		Time:  time.Now().Unix(),
		Size:  fileSize,
	}
	copy(meta.Name[:], nameBase)

	var firstChunk string
	for i := 0; ; i++ {
		n, err := file.Read(buf)
		if n > 0 {
			chunkName := fmt.Sprintf("%s_%04d.part", strings.TrimSuffix(nameBase, filepath.Ext(nameBase)), i)
			fullPath := filepath.Join(outDir, chunkName)
			if i == 0 {
				fullPath = strings.Replace(fullPath, "part", "tmp", 1)
				firstChunk = fullPath
			}
			if writeErr := os.WriteFile(fullPath, buf[:n], 0644); writeErr != nil {
				return writeErr
			}
			hash.Write(buf[:n])
		}
		if err != nil {
			if err == io.EOF {
				copy(meta.Hash[:], hash.Sum(nil))
				return s.injectMetadata(firstChunk, &meta)
			}
			return err
		}
	}
}

func (s *Split) MergeFile(inDir string) error {
	chunks, err := s.checkFiles(inDir)
	if err != nil {
		return err
	}

	var meta metadata
	for _, c := range chunks {
		if c.first {
			if err := s.extractMetadata(c.name, &meta); err != nil {
				return err
			}
			break
		}
	}

	outFile, err := os.Create(filepath.Join(inDir, string(bytes.Trim(meta.Name[:], "\x00"))))
	if err != nil {
		return err
	}
	defer outFile.Close()

	hash := sha256.New()

	for _, chunk := range chunks {
		f, err := os.Open(chunk.name)
		if err != nil {
			return err
		}

		if chunk.first {
			if _, err := f.Seek(int64(binary.Size(meta)), io.SeekStart); err != nil {
				f.Close()
				return err
			}
		}

		if _, err := io.Copy(outFile, io.TeeReader(f, hash)); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}

	if !bytes.Equal(hash.Sum(nil), meta.Hash[:]) {
		return errors.New("hash mismatch: file not reconstructed properly")
	}

	for _, c := range chunks {
		_ = os.Remove(c.name)
	}

	fmt.Println("Merge successful.")
	return nil
}

func (s *Split) SplitData(v any, a []any, chunks int) error {
	if v == nil {
		return errors.New("input is nil")
	}
	if chunks < 2 {
		return errors.New("chunks must be at least 2")
	}
	if len(a) != chunks {
		return fmt.Errorf("output slice length must be %d", chunks)
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return fmt.Errorf("gob encode failed: %w", err)
	}
	blob := buf.Bytes()

	partSize := len(blob) / chunks
	if partSize == 0 {
		partSize = 1
	}

	for i := 0; i < chunks; i++ {
		start := i * partSize
		end := start + partSize
		if i == chunks-1 || end > len(blob) {
			end = len(blob)
		}
		a[i] = blob[start:end]
	}

	return nil
}

func (s *Split) MergeData(a []any, v any) error {
	if v == nil {
		return errors.New("output is nil")
	}
	var combined []byte
	for _, part := range a {
		b, ok := part.([]byte)
		if !ok {
			return fmt.Errorf("chunk type is not []byte")
		}
		combined = append(combined, b...)
	}
	return gob.NewDecoder(bytes.NewReader(combined)).Decode(v)
}

func (s *Split) encodeFormat(v any, format string) ([]byte, error) {
	var buf bytes.Buffer
	switch strings.ToLower(format) {
	case "gob":
		if err := gob.NewEncoder(&buf).Encode(v); err != nil {
			return nil, err
		}
	case "json":
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	return buf.Bytes(), nil
}

func (s *Split) decodeFormat(data []byte, v any, format string) error {
	switch strings.ToLower(format) {
	case "gob":
		return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
	case "json":
		return json.Unmarshal(data, v)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

type parsedChunk struct {
	first bool
	name  string
	index int
}

func (s *Split) injectMetadata(chunkPath string, meta *metadata) error {
	src, err := os.Open(chunkPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dstName := strings.Replace(chunkPath, "tmp", "part", 1)
	dst, err := os.Create(dstName)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, meta); err != nil {
		return err
	}

	if _, err := dst.Write(buf.Bytes()); err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return os.Remove(chunkPath)
}

func (s *Split) extractMetadata(filePath string, meta *metadata) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	return binary.Read(f, binary.BigEndian, meta)
}

func (s *Split) checkFiles(dir string) ([]parsedChunk, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var chunks []parsedChunk
	re := regexp.MustCompile(`_(\d{4})\.part$`)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := filepath.Join(dir, e.Name())
		m := re.FindStringSubmatch(e.Name())
		if len(m) != 2 {
			continue
		}
		var idx int
		_, _ = fmt.Sscanf(m[1], "%d", &idx)
		chunks = append(chunks, parsedChunk{name: name, index: idx, first: idx == 0})
	}

	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].index < chunks[j].index
	})

	return chunks, nil
}
