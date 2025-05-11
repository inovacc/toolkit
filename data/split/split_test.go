package split

import (
	"os"
	"path/filepath"
	"testing"
)

const testDataDir = "testdata"

func TestNewSplit(t *testing.T) {
	s := NewSplit()
	file, err := os.Open(filepath.Join(testDataDir, "night.city_cars.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SplitFile(file, filepath.Join(testDataDir, "output"), 5); err != nil {
		t.Fatal(err)
	}

	if err := s.MergeFile(filepath.Join(testDataDir, "output")); err != nil {
		t.Fatal(err)
	}
}
