package glitch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/inovacc/toolkit/data/algorithm/hashing"
)

func TestGlitchEncodeDecode(t *testing.T) {
	g := NewGlitch()

	// Setup test directories and files
	testInput := "/home/dyam/Downloads/gocv-0.41.0.zip"
	testOutputDir := "testdata/frames"
	testReconstructed := "testdata/reconstructed/gocv-0.41.0.zip"

	// defer os.RemoveAll(testOutputDir)
	// defer os.RemoveAll(testReconstructed)

	// encode a file to images
	if err := g.EncodeFileToImages(testInput, testOutputDir); err != nil {
		t.Fatalf("EncodeFileToImages failed: %v", err)
	}

	// Decode images back to file
	pattern := filepath.Join(testOutputDir, "frame_*.png")
	meta, err := g.DecodeImagesToFile(pattern, testReconstructed)
	if err != nil {
		t.Fatalf("DecodeImagesToFile failed: %v", err)
	}

	// Validate content
	got, err := os.ReadFile(filepath.Join(testReconstructed, meta.Name))
	if err != nil {
		t.Fatalf("Failed to read reconstructed file: %v", err)
	}

	newHasher := hashing.NewHasher(hashing.SHA256)
	gotStr := newHasher.HashBytes(got)

	source, err := os.ReadFile(testInput)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	sourceStr := newHasher.HashBytes(source)

	if gotStr != sourceStr {
		t.Fatal("not equal")
	}

	if err := g.MakeVideo(testOutputDir, "testdata", false); err != nil {
		t.Fatalf("MakeVideo failed: %v", err)
	}

	videoPath := "testdata/output.mkv"
	tempFramesDir := "testdata/temp_frames"
	outputDir := "testdata/decoded"

	meta, err = g.ExtractFileFromVideo(videoPath, tempFramesDir, outputDir)
	if err != nil {
		log.Fatalf("Failed to extract file from video: %v", err)
	}

	fmt.Printf("Successfully extracted: %s\nCreated: %v\nHash: %x\n", meta.Name, meta.Date, meta.Hash)
}
