package mnemonic

import (
	"testing"
)

func TestSeed_ToHex(t *testing.T) {
	s := &Seed{Bytes: []byte{0xDE, 0xAD, 0xBE, 0xEF}}
	want := "deadbeef"
	if got := s.ToHex(); got != want {
		t.Errorf("ToHex() = %v, want %v", got, want)
	}
}

func TestSeed_String(t *testing.T) {
	s := &Seed{Bytes: []byte("abc")}
	if got := s.String(); got != s.ToHex() {
		t.Errorf("String() != ToHex()")
	}
}
