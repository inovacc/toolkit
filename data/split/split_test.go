package split

import (
	"fmt"
	"log"
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

type MyStruct struct {
	UserID string
	Values []int
}

func TestNewSplitData(t *testing.T) {
	s := NewSplit()

	input := MyStruct{
		UserID: "john",
		Values: []int{10, 20, 30, 40, 50},
	}
	chunks := make([]any, 3)

	if err := s.SplitData(input, chunks, 3); err != nil {
		log.Fatalf("split error: %v", err)
	}

	var output MyStruct
	if err := s.MergeData(chunks, &output); err != nil {
		log.Fatalf("merge error: %v", err)
	}

	fmt.Printf("Restored: %+v\n", output)
}
