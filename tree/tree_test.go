package tree

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/spf13/afero"
)

func TestNewTree(t *testing.T) {
	fs := afero.NewOsFs()

	filePath := "testdata"

	tree := NewTree(fs, filePath, NewConfig(WithExclude("mock", ".git", "node_modules", "target", ".idea")))

	if err := tree.MakeTree(); err != nil {
		t.Fatalf("Failed to build tree: %v", err)
	}

	actual := []byte(tree.ToString())

	exists, err := afero.Exists(fs, filePath)
	if err != nil || !exists {
		t.Fatalf("Expected file does not exist: %s", filePath)
	}

	goldenPath := "./testdata.golden"

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Error reading golden file: %s\n%v", goldenPath, err)
	}

	if err := compareContent(actual, expected); err != nil {
		t.Fatalf("Mismatch for %s:\n%v", filePath, err)
	}
}

func compareContent(contentA, contentB []byte) error {
	if !bytes.Equal(ensureLF(contentA), ensureLF(contentB)) {
		return errors.New("byte slices differ")
	}
	return nil
}

// ensureLF converts any \r\n to \n
func ensureLF(content []byte) []byte {
	return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
}
