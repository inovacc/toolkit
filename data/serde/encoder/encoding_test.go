package encoder

import (
	"log"
	"os"
	"testing"

	"github.com/inovacc/toolkit/data/algorithm/random"
)

func TestBase02Encode(t *testing.T) {
	testEncoding := NewEncoding(Base02)

	str := "hello world"
	encoded, err := testEncoding.EncodeStr(str)
	if err != nil {
		log.Fatal(err)
	}

	if encoded != "0110100001100101011011000110110001101111001000000111011101101111011100100110110001100100" {
		t.Fatalf("Expected %s, got %s", "0110100001100101011011000110110001101111001000000111011101101111011100100110110001100100", encoded)
	}
}

func TestBase58Encode(t *testing.T) {
	testEncoding := NewEncoding(Base58)

	str := "hello world"
	encoded, err := testEncoding.EncodeStr(str)
	if err != nil {
		log.Fatal(err)
	}

	if encoded != "StV1DL6CwTryKyV" {
		t.Fatalf("Expected %s, got %s", "StV1DL6CwTryKyV", encoded)
	}
}

func TestBase58Decode(t *testing.T) {
	testEncoding := NewEncoding(Base58)

	data := []byte("Hello, World!")
	encoded, err := testEncoding.Encode(data)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err := testEncoding.Decode(encoded)
	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	if string(decoded) != string(data) {
		t.Fatalf("Decoded string does not match original: got %s, want %s", decoded, data)
	}
}

func TestBase62Encode(t *testing.T) {
	testEncoding := NewEncoding(Base62)

	str := "hello world"
	encoded, err := testEncoding.EncodeStr(str)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the encoded string is not empty
	if encoded != "AAwf93rvy4aWQVw" {
		t.Fatalf("Expected %s, got %s", "AAwf93rvy4aWQVw", encoded)
	}

	// Check if the encoded string is a valid base62 string
	for _, char := range encoded {
		if (char < '0' || char > '9') && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') {
			t.Fatalf("Encoded string contains invalid character: %c", char)
		}
	}
}

func TestBase62Decode(t *testing.T) {
	testEncoding := NewEncoding(Base62)

	data := []byte("Hello, World!")
	encoded, err := testEncoding.Encode(data)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err := testEncoding.Decode(encoded)
	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	if string(decoded) != string(data) {
		t.Fatalf("Decoded string does not match original: got %s, want %s", decoded, data)
	}
}

func TestBase64Encode(t *testing.T) {
	testEncoding := NewEncoding(Base64)

	str := "hello world"
	encoded, err := testEncoding.EncodeStr(str)
	if err != nil {
		log.Fatal(err)
	}

	if encoded != "aGVsbG8gd29ybGQ=" {
		t.Fatalf("Expected %s, got %s", "aGVsbG8gd29ybGQ=", encoded)
	}
}

func TestBase64Decode(t *testing.T) {
	testEncoding := NewEncoding(Base64)

	data := []byte("Hello, World!")
	encoded, err := testEncoding.Encode(data)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err := testEncoding.Decode(encoded)
	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	if string(decoded) != string(data) {
		t.Fatalf("Decoded string does not match original: got %s, want %s", decoded, data)
	}
}

func TestEncodeDecodeStrPortable(t *testing.T) {
	str := random.RandomString(5000)

	testEncoding := NewEncoding(Base64)
	testEncoding.SetLimit(100)

	encoded, err := testEncoding.EncodeStr(str)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err := testEncoding.DecodeStr(encoded)
	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	if decoded != str {
		t.Fatalf("Decoded string does not match original: got %s, want %s", decoded, str)
	}

	testEncoding = NewEncoding(Base62)
	testEncoding.SetLimit(100)

	encoded, err = testEncoding.EncodeStr(str)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err = testEncoding.DecodeStr(encoded)
	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	if decoded != str {
		t.Fatalf("Decoded string does not match original: got %s, want %s", decoded, str)
	}

	testEncoding = NewEncoding(Base58)
	testEncoding.SetLimit(100)

	encoded, err = testEncoding.EncodeStr(str)
	if err != nil {
		log.Fatal(err)
	}

	decoded, err = testEncoding.DecodeStr(encoded)
	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	if decoded != str {
		t.Fatalf("Decoded string does not match original: got %s, want %s", decoded, str)
	}
}

func TestEncodeDecodeStrPortableFile(t *testing.T) {
	data, err := os.ReadFile("testdata/file.txt")
	if err != nil {
		t.Fatal(err)
	}

	testEncoding := NewEncoding(Base64)
	testEncoding.SetLimit(180)

	decoded, err := testEncoding.DecodeStr(string(data))
	if err != nil {
		t.Fatalf("Error decoding string: %v", err)
	}

	str := ""

	if decoded != str {
		t.Fatalf("Decoded string does not match original: got %s, want %s", decoded, str)
	}
}
