package split

import (
	"os"
	"path/filepath"
	"testing"
)

const testDataDir = "testdata"

func TestNewSplit(t *testing.T) {
	s := NewSplit()
	file, err := os.Open(filepath.Join(testDataDir, "ubuntu-25.04-desktop-amd64.iso"))
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SplitFile(file, filepath.Join(testDataDir, "output"), 50); err != nil {
		t.Fatal(err)
	}

	if err := s.MergeFile(filepath.Join(testDataDir, "output")); err != nil {
		t.Fatal(err)
	}
}
