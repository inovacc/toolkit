// Package split provides functionality for splitting and merging files and data structures.
// It allows breaking down large files into smaller chunks for easier transmission,
// storage, or processing, and then reconstructing them later.
package split

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
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

// Constants for file operations
const (
	// DefaultFilePermissions is the default permission for created files
	DefaultFilePermissions = 0644

	// DefaultDirPermissions is the default permission for created directories
	DefaultDirPermissions = 0755

	// MinChunks is the minimum number of chunks required for splitting
	MinChunks = 2

	// MaxFilenameLength is the maximum length of a filename in the metadata
	MaxFilenameLength = 46
)

// metadata stores essential information about the split file
type metadata struct {
	Hash  [32]byte                // 32 bytes SHA-256
	Total uint32                  // 4 bytes
	Size  int64                   // 8 bytes
	Time  int64                   // 8 bytes
	Name  [MaxFilenameLength]byte // truncated or padded filename
}

// Split is a utility struct for splitting and merging files and data
type Split struct{}

// NewSplit creates a new instance of the Split utility
func NewSplit() *Split {
	return &Split{}
}

// SplitFile splits a file into multiple chunks of roughly equal size.
// It creates chunks in the specified output directory and adds metadata to the first chunk.
// The metadata includes an SHA-256 hash of the original file, which is used to verify
// data integrity during merging.
//
// Parameters:
//   - file: Pointer to the file to split
//   - outDir: Directory to store the chunks
//   - chunks: Number of chunks to create (minimum 2)
//
// Returns an error if any part of the process fails.
func (s *Split) SplitFile(file *os.File, outDir string, chunks int) error {
	if chunks < MinChunks {
		return fmt.Errorf("chunks must be at least %d", MinChunks)
	}

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stats: %w", err)
	}

	fileSize := stat.Size()
	chunkSize := fileSize/int64(chunks) + 1
	buf := make([]byte, chunkSize)

	if err := os.MkdirAll(outDir, DefaultDirPermissions); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
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
			if writeErr := os.WriteFile(fullPath, buf[:n], DefaultFilePermissions); writeErr != nil {
				return fmt.Errorf("failed to write chunk file: %w", writeErr)
			}
			hash.Write(buf[:n])
		}
		if err != nil {
			if err == io.EOF {
				copy(meta.Hash[:], hash.Sum(nil))
				return s.injectMetadata(firstChunk, &meta)
			}
			return fmt.Errorf("error reading file: %w", err)
		}
	}
}

// MergeFile reconstructs a file from its chunks in the specified directory.
// It extracts metadata from the first chunk, combines all chunks into a single file,
// and verifies the SHA-256 hash to ensure data integrity.
// After successful merging, it removes the chunk files.
//
// Parameters:
//   - inDir: Directory containing the chunks
//
// Returns an error if any part of the process fails.
func (s *Split) MergeFile(inDir string) error {
	chunks, err := s.checkFiles(inDir)
	if err != nil {
		return fmt.Errorf("failed to check chunk files: %w", err)
	}

	if len(chunks) == 0 {
		return errors.New("no chunk files found in the specified directory")
	}

	// Extract metadata from the first chunk
	var meta metadata
	var foundFirstChunk bool
	for _, c := range chunks {
		if c.first {
			if err := s.extractMetadata(c.name, &meta); err != nil {
				return fmt.Errorf("failed to extract metadata: %w", err)
			}
			foundFirstChunk = true
			break
		}
	}

	if !foundFirstChunk {
		return errors.New("first chunk (index 0) not found")
	}

	// Create an output file
	outputFileName := string(bytes.Trim(meta.Name[:], "\x00"))
	outFile, err := os.Create(filepath.Join(inDir, outputFileName))
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			// We can only log the error since we're in a deferred
			fmt.Printf("Error closing output file: %v\n", closeErr)
		}
	}()

	hash := sha256.New()

	// Process each chunk
	for _, chunk := range chunks {
		f, err := os.Open(chunk.name)
		if err != nil {
			return fmt.Errorf("failed to open chunk file %s: %w", chunk.name, err)
		}

		// Skip metadata in the first chunk
		if chunk.first {
			if _, err := f.Seek(int64(binary.Size(meta)), io.SeekStart); err != nil {
				return fmt.Errorf("failed to seek past metadata: %w", err)
			}
		}

		// Copy chunk data to an output file and calculate hash
		if _, err := io.Copy(outFile, io.TeeReader(f, hash)); err != nil {
			// Close the file before returning the error
			_ = f.Close() // Ignore the close error since we're already handling another error
			return fmt.Errorf("failed to copy chunk data: %w", err)
		}

		// Close the file explicitly after processing to release resources immediately
		// This is better than using defer inside a loop which would accumulate open files
		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close chunk file: %w", err)
		}
	}

	// Verify data integrity
	if !bytes.Equal(hash.Sum(nil), meta.Hash[:]) {
		return errors.New("hash mismatch: file not reconstructed properly")
	}

	// Remove chunk files after a successful merge
	for _, c := range chunks {
		if err := os.Remove(c.name); err != nil {
			fmt.Printf("Warning: failed to remove chunk file %s: %v\n", c.name, err)
		}
	}

	fmt.Printf("Merge successful. File saved as: %s\n", outputFileName)
	return nil
}

// SplitData splits arbitrary Go data into chunks.
// It encodes the data using gob encoding and splits the encoded bytes into roughly equal chunks.
//
// Parameters:
//   - v: Data to split (any type)
//   - a: Slice to store the chunks (must be pre-allocated with length equal to chunks)
//   - chunks: Number of chunks to create (minimum 2)
//
// Returns an error if any part of the process fails.
func (s *Split) SplitData(v any, a []any, chunks int) error {
	if v == nil {
		return errors.New("input is nil")
	}
	if chunks < MinChunks {
		return fmt.Errorf("chunks must be at least %d", MinChunks)
	}
	if len(a) != chunks {
		return fmt.Errorf("output slice length must be %d", chunks)
	}

	// Encode the data using gob encoding
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return fmt.Errorf("gob encode failed: %w", err)
	}
	encodedData := buf.Bytes()

	// Calculate chunk size
	dataLength := len(encodedData)
	partSize := dataLength / chunks
	if partSize == 0 && dataLength > 0 {
		partSize = 1
	}

	// Split the encoded data into chunks
	for i := 0; i < chunks; i++ {
		start := i * partSize
		end := start + partSize

		// Ensure the last chunk includes any remaining bytes
		if i == chunks-1 || end > dataLength {
			end = dataLength
		}

		// Skip empty chunks if data is smaller than the number of chunks
		if start >= dataLength {
			a[i] = []byte{}
		} else {
			a[i] = encodedData[start:end]
		}
	}

	return nil
}

// MergeData reconstructs data from chunks.
// It combines all chunks into a single byte slice and decodes it using gob decoding.
//
// Parameters:
//   - a: Slice containing the chunks
//   - v: Pointer to store the reconstructed data
//
// Returns an error if any part of the process fails.
func (s *Split) MergeData(a []any, v any) error {
	if v == nil {
		return errors.New("output is nil")
	}

	if len(a) == 0 {
		return errors.New("no chunks provided")
	}

	// Combine all chunks into a single byte slice
	var combined []byte
	for i, part := range a {
		b, ok := part.([]byte)
		if !ok {
			return fmt.Errorf("chunk at index %d is not []byte", i)
		}
		combined = append(combined, b...)
	}

	// Decode the combined data
	if len(combined) == 0 {
		return errors.New("no data to decode")
	}

	return gob.NewDecoder(bytes.NewReader(combined)).Decode(v)
}

// parsedChunk represents a chunk file with its metadata
type parsedChunk struct {
	first bool   // indicates if this is the first chunk (contains metadata)
	name  string // full path to the chunk file
	index int    // numerical index of the chunk
}

// injectMetadata adds metadata to the first chunk.
// It creates a new file with metadata at the beginning, followed by the chunk data.
// The original temporary file is removed after a successful operation.
func (s *Split) injectMetadata(chunkPath string, meta *metadata) error {
	src, err := os.Open(chunkPath)
	if err != nil {
		return fmt.Errorf("failed to open source chunk file: %w", err)
	}
	defer func(src *os.File) {
		if err := src.Close(); err != nil {
			fmt.Printf("Error closing source file: %v\n", err)
		}
	}(src)

	// Create a destination file with .part extension
	dstName := strings.Replace(chunkPath, "tmp", "part", 1)
	dst, err := os.Create(dstName)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func(dst *os.File) {
		if err := dst.Close(); err != nil {
			fmt.Printf("Error closing destination file: %v\n", err)
		}
	}(dst)

	// Write metadata to a buffer
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, meta); err != nil {
		return fmt.Errorf("failed to write metadata to buffer: %w", err)
	}

	// Write metadata to a destination file
	if _, err := dst.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("failed to write metadata to file: %w", err)
	}

	// Copy chunk data to a destination file
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy chunk data: %w", err)
	}

	// Remove a temporary file
	if err := os.Remove(chunkPath); err != nil {
		return fmt.Errorf("failed to remove temporary file: %w", err)
	}

	return nil
}

// extractMetadata retrieves metadata from the first chunk.
// It reads the binary metadata structure from the beginning of the file.
func (s *Split) extractMetadata(filePath string, meta *metadata) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for metadata extraction: %w", err)
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing file: %v\n", err)
		}
	}(f)

	if err := binary.Read(f, binary.BigEndian, meta); err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	return nil
}

// checkFiles identifies and sorts chunk files in a directory.
// It uses regex to find files with the pattern `_NNNN.part`.
func (s *Split) checkFiles(dir string) ([]parsedChunk, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
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
		if _, scanErr := fmt.Sscanf(m[1], "%d", &idx); scanErr != nil {
			fmt.Printf("Warning: failed to parse chunk index from %s: %v\n", e.Name(), scanErr)
			continue
		}

		chunks = append(chunks, parsedChunk{name: name, index: idx, first: idx == 0})
	}

	// Sort chunks by index
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].index < chunks[j].index
	})

	return chunks, nil
}
