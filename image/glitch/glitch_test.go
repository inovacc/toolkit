package glitch

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/inovacc/toolkit/data/algorithm/hashing"
)

const testDataDir = "../testdata"

func TestGlitchEncodeDecode(t *testing.T) {
	g := NewGlitch()

	// Setup test directories and files
	testInput := filepath.Join(testDataDir, "lena_original.jpg")
	testOutputDir := filepath.Join(testDataDir, "frames")
	testReconstructed := filepath.Join(testDataDir, "reconstructed/lena_original.jpg")

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

	videoPath := filepath.Join(testDataDir, "output.mkv")
	tempFramesDir := filepath.Join(testDataDir, "temp_frames")
	outputDir := filepath.Join(testDataDir, "decoded")

	meta, err = g.ExtractFileFromVideo(videoPath, tempFramesDir, outputDir)
	if err != nil {
		log.Fatalf("Failed to extract file from video: %v", err)
	}

	fmt.Printf("Successfully extracted: %s\nCreated: %v\nHash: %x\n", meta.Name, meta.Date, meta.Hash)
}
