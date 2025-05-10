package fileutil

import (
	"bytes"
	"os"
	"testing"
)

var (
	testData        = []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod")
	filename        = "testfile.txt"
	nonExistentFile = "non_existent_file.txt"
)

func TestWriteToFile(t *testing.T) {
	if err := WriteToFile(filename, testData); err != nil {
		t.Errorf("WriteToFile failed: %v", err)
		return
	}

	if err := os.Remove(filename); err != nil {
		t.Errorf("Remove failed: %v", err)
		return
	}
}

func TestReadFromFile(t *testing.T) {
	if err := WriteToFile(filename, testData); err != nil {
		t.Errorf("WriteToFile failed: %v", err)
		return
	}

	data, err := ReadFromFile(filename)
	if err != nil {
		t.Errorf("ReadFromFile failed: %v", err)
		return
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("ReadFromFile returned incorrect data: %s", data)
		return
	}

	if err := os.Remove(filename); err != nil {
		t.Errorf("Remove failed: %v", err)
		return
	}
}

func TestReadFromFileError(t *testing.T) {
	_, err := ReadFromFile(nonExistentFile)
	if err == nil {
		t.Errorf("Expected error when reading from non-existent file, got nil")
		return
	}
}
