package random

import (
	"testing"
)

func TestRandomString(t *testing.T) {
	v := RandomString(20)
	if v == "" {
		t.Error("string is empty")
		return
	}
}

func TestRandomInt(t *testing.T) {
	v, err := RandomInt(0, 30)
	if err != nil {
		t.Error(err)
		return
	}

	if v < 0 {
		t.Error("number is not valid")
		return
	}
}

func TestRandomBytes(t *testing.T) {
	v, err := RandomBytes(20)
	if err != nil {
		t.Error(err)
		return
	}

	if v == nil {
		t.Error("bytes are empty")
		return
	}
}
