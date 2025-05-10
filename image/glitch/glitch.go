package glitch

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	brightnessThreshold = 0x8000
	pixelSize           = 4
	width               = 1920
	height              = 1080
)

type FileMetadata struct {
	Name string
	Date int64
	Hash [32]byte
}

type Glitch struct {
	comp *compression.Compress
}

func NewGlitch() *Glitch {
	return &Glitch{
		comp: compression.NewCompress(compression.TypeGzip),
	}
}

func (g *Glitch) DecodeImagesToFile(imagesPath, outputDir string) (*FileMetadata, error) {
	if err := g.ensureDir(outputDir); err != nil {
		return nil, err
	}

	binaryStr, err := g.extractBinaryString(imagesPath)
	if err != nil {
		return nil, err
	}

	data, err := g.decodeBinaryPayload(binaryStr)
	if err != nil {
		return nil, err
	}

	meta, fileData, err := g.parseAndDecompress(data, g.comp)
	if err != nil {
		return nil, err
	}

	if err := g.validateHash(fileData, meta.Hash); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(outputDir, meta.Name), fileData, 0644); err != nil {
		return meta, err
	}
	return meta, nil
}

func (g *Glitch) EncodeFileToImages(filename, outputDir string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	meta := FileMetadata{
		Name: filepath.Base(filename),
		Date: time.Now().Unix(),
		Hash: sha256.Sum256(data),
	}

	metaBuf := new(bytes.Buffer)
	if err := binary.Write(metaBuf, binary.BigEndian, int32(len(meta.Name))); err != nil {
		return err
	}

	if _, err := metaBuf.WriteString(meta.Name); err != nil {
		return err
	}

	if err := binary.Write(metaBuf, binary.BigEndian, meta.Date); err != nil {
		return err
	}

	if _, err := metaBuf.Write(meta.Hash[:]); err != nil {
		return err
	}

	compressed, err := g.comp.Compress(data)
	if err != nil {
		return err
	}

	fullPayload := append(metaBuf.Bytes(), compressed...)

	bitLen := uint32(len(fullPayload) * 8)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, bitLen)
	prefixed := append(lenBuf, fullPayload...)

	var buf bytes.Buffer
	for _, b := range prefixed {
		buf.WriteString(fmt.Sprintf("%08b", b))
	}
	binaryStr := buf.String()

	pixelsPerImage := (width / pixelSize) * (height / pixelSize)
	numImages := int(math.Ceil(float64(len(binaryStr)) / float64(pixelsPerImage)))

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	var wg sync.WaitGroup
	errCh := make(chan error, numImages)

	for i := 0; i < numImages; i++ {
		wg.Add(1)
		var metaRef *FileMetadata
		if i == numImages-1 {
			metaRef = &meta
		}
		go g.worker(i, binaryStr, outputDir, pixelsPerImage, numImages, errCh, &wg, metaRef)
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh
	}
	return nil
}

func (g *Glitch) safeOpen(file string) (image.Image, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}(f)

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("image decode failed for %s: %w", file, err)
	}
	return img, nil
}

func (g *Glitch) worker(i int, binaryStr, outputDir string, pixelsPerImage, numImages int, errCh chan error, wg *sync.WaitGroup, meta *FileMetadata) {
	defer wg.Done()

	start := i * pixelsPerImage
	end := start + pixelsPerImage
	if end > len(binaryStr) {
		end = len(binaryStr)
	}
	subStr := binaryStr[start:end]
	img := image.NewGray(image.Rect(0, 0, width, height))

	idx := 0
	startY := 0
	if i == numImages-1 && meta != nil {
		startY = pixelSize // skip metadata row
	}
	for y := startY; y < height; y += pixelSize {
		if i == numImages-1 && meta != nil {
			countBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(countBuf, uint32(numImages))
			for k := 0; k < 4; k++ {
				img.SetGray(k, 0, color.Gray{Y: countBuf[k]})
			}

			metaBuf := new(bytes.Buffer)
			_ = binary.Write(metaBuf, binary.BigEndian, int32(len(meta.Name)))
			metaBuf.WriteString(meta.Name)
			_ = binary.Write(metaBuf, binary.BigEndian, meta.Date)
			metaBuf.Write(meta.Hash[:])

			metaBytes := metaBuf.Bytes()
			for i, b := range metaBytes {
				img.SetGray(i+4, 0, color.Gray{Y: b})
			}
		}

		for x := 0; x < width; x += pixelSize {
			if idx >= len(subStr) {
				break
			}
			c := color.White
			if subStr[idx] == '1' {
				c = color.Black
			}
			for dy := 0; dy < pixelSize; dy++ {
				for dx := 0; dx < pixelSize; dx++ {
					img.Set(x+dx, y+dy, c)
				}
			}
			idx++
		}
	}

	outPath := filepath.Join(outputDir, fmt.Sprintf("frame_%04d.png", i))
	outFile, err := os.Create(outPath)
	if err != nil {
		errCh <- err
		return
	}

	if i == numImages-1 {
		// Store numImages as 4 bytes in top-left 4 pixels
		countBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(countBuf, uint32(numImages))

		for k := 0; k < 4; k++ {
			img.SetGray(k, 0, color.Gray{Y: countBuf[k]})
		}
	}

	err = png.Encode(outFile, img)
	if cerr := outFile.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		errCh <- err
	}
}

func (g *Glitch) ensureDir(path string) error {
	if _, err := os.Stat(path); err != nil {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

func (g *Glitch) extractBinaryString(imagesPath string) (string, error) {
	files, err := filepath.Glob(imagesPath)
	if err != nil || len(files) == 0 {
		return "", errors.New("no image frames found")
	}
	sort.Strings(files)

	lastImg, err := g.safeOpen(files[len(files)-1])
	if err != nil {
		return "", fmt.Errorf("failed to open last frame: %w", err)
	}

	grayImg, ok := lastImg.(*image.Gray)
	if !ok {
		return "", fmt.Errorf("expected grayscale image: %s", files[len(files)-1])
	}

	var frameCountBytes [4]byte
	for k := 0; k < 4; k++ {
		frameCountBytes[k] = grayImg.GrayAt(k, 0).Y
	}
	totalFrames := binary.BigEndian.Uint32(frameCountBytes[:])

	// Read metadata from next bytes in row
	var nameLen int32
	metaBuf := make([]byte, 4)
	for i := 0; i < 4; i++ {
		metaBuf[i] = grayImg.GrayAt(i+4, 0).Y
	}
	_ = binary.Read(bytes.NewReader(metaBuf), binary.BigEndian, &nameLen)

	nameBytes := make([]byte, nameLen)
	for i := int32(0); i < nameLen; i++ {
		nameBytes[i] = grayImg.GrayAt(int(8+i), 0).Y
	}
	name := string(nameBytes)

	timeBuf := make([]byte, 8)
	for i := 0; i < 8; i++ {
		timeBuf[i] = grayImg.GrayAt(int(8+nameLen)+i, 0).Y
	}
	var timestamp int64
	_ = binary.Read(bytes.NewReader(timeBuf), binary.BigEndian, &timestamp)

	hash := [32]byte{}
	for i := 0; i < 32; i++ {
		hash[i] = grayImg.GrayAt(int(8+nameLen+8)+i, 0).Y
	}

	meta := FileMetadata{
		Name: name,
		Date: timestamp,
		Hash: hash,
	}
	fmt.Printf("Extracted metadata: %+v\nDetected total frames: %d\n", meta, totalFrames)

	var buf bytes.Buffer
	for _, file := range files {
		img, err := g.safeOpen(file)
		if err != nil {
			return "", err
		}
		startY := 0
		if file == files[len(files)-1] {
			startY = pixelSize // skip metadata row
		}
		for y := startY; y < height; y += pixelSize {
			for x := 0; x < width; x += pixelSize {
				r, g, b, _ := img.At(x, y).RGBA()
				avg := (r + g + b) / 3
				if avg < brightnessThreshold {
					buf.WriteByte('1')
				} else {
					buf.WriteByte('0')
				}
			}
		}
	}
	return g.trimPaddedBits(buf.String())
}

func (g *Glitch) trimPaddedBits(binaryStr string) (string, error) {
	if len(binaryStr) < 32 {
		return "", errors.New("binary string too short to contain length header")
	}

	var totalBits uint32
	if _, err := fmt.Sscanf(binaryStr[:32], "%032b", &totalBits); err != nil {
		return "", fmt.Errorf("failed to parse bit length prefix: %w", err)
	}

	if len(binaryStr) < int(32+totalBits) {
		return "", fmt.Errorf("binary string is shorter than declared bit length: expected %d bits, got %d", totalBits, len(binaryStr)-32)
	}

	trimmed := binaryStr[32 : 32+totalBits]
	if len(trimmed)%8 != 0 {
		return "", errors.New("trimmed binary string is not byte aligned")
	}
	return trimmed, nil
}

func (g *Glitch) decodeBinaryPayload(binaryStr string) ([]byte, error) {
	data := make([]byte, len(binaryStr)/8)
	for i := range data {
		if _, err := fmt.Sscanf(binaryStr[i*8:(i+1)*8], "%08b", &data[i]); err != nil {
			return nil, fmt.Errorf("binary decode failed at index %d: %w", i, err)
		}
	}
	return data, nil
}

func (g *Glitch) parseAndDecompress(data []byte, comp *compression.Compress) (*FileMetadata, []byte, error) {
	meta := &FileMetadata{}
	r := bytes.NewReader(data)

	var nameLen int32
	if err := binary.Read(r, binary.BigEndian, &nameLen); err != nil {
		return meta, nil, fmt.Errorf("failed to read name length: %w", err)
	}

	nameBytes := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBytes); err != nil {
		return meta, nil, fmt.Errorf("failed to read name: %w", err)
	}
	meta.Name = string(nameBytes)

	if err := binary.Read(r, binary.BigEndian, &meta.Date); err != nil {
		return meta, nil, fmt.Errorf("failed to read timestamp: %w", err)
	}

	if _, err := io.ReadFull(r, meta.Hash[:]); err != nil {
		return meta, nil, fmt.Errorf("failed to read hash: %w", err)
	}

	compressedData, err := io.ReadAll(r)
	if err != nil {
		return meta, nil, fmt.Errorf("failed to read file content: %w", err)
	}

	decompressed, err := comp.Decompress(compressedData)
	if err != nil {
		return meta, nil, fmt.Errorf("failed to decompress: %w", err)
	}

	return meta, decompressed, nil
}

func (g *Glitch) validateHash(data []byte, expected [32]byte) error {
	actual := sha256.Sum256(data)
	if actual != expected {
		return fmt.Errorf("hash mismatch: file may be corrupted")
	}
	return nil
}

func (g *Glitch) MakeVideo(imagesPath, outputDir string, compress bool) error {
	outputFile := filepath.Join(outputDir, "output.mkv")

	var cmd *exec.Cmd
	if compress {
		cmd = exec.Command("ffmpeg",
			"-framerate", "30",
			"-pattern_type", "glob",
			"-i", filepath.Join(imagesPath, "frame_*.png"),
			"-c:v", "libaom-av1",
			"-crf", "30",
			"-b:v", "0",
			outputFile,
		)
	} else {
		cmd = exec.Command("ffmpeg",
			"-framerate", "30",
			"-pattern_type", "glob",
			"-i", filepath.Join(imagesPath, "frame_*.png"),
			"-c:v", "libx265",
			"-crf", "28",
			outputFile,
		)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("command: %s\n", cmd.String())

	fmt.Println("Generating video...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	fmt.Println("Video created:", outputFile)
	return nil
}

func (g *Glitch) ExtractFramesFromVideo(videoPath, framesDir string) error {
	if err := g.ensureDir(framesDir); err != nil {
		return fmt.Errorf("failed to create frames directory: %w", err)
	}

	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vsync", "0",
		"-frame_pts", "1",
		filepath.Join(framesDir, "frame_%04d.png"),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Extracting frames from video: %s\n", cmd.String())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}
	return nil
}

func (g *Glitch) ExtractFileFromVideo(videoPath, tempDir, outputDir string) (*FileMetadata, error) {
	if err := g.ExtractFramesFromVideo(videoPath, tempDir); err != nil {
		return nil, err
	}
	return g.DecodeImagesToFile(filepath.Join(tempDir, "*.png"), outputDir)
}
