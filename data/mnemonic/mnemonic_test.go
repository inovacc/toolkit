package mnemonic

import (
	"testing"

	"github.com/inovacc/toolkit/data/mnemonic/entropy"
)

func TestNewRandom(t *testing.T) {
	m, err := NewRandom(128, English)
	if err != nil {
		t.Fatalf("NewRandom() error: %v", err)
	}
	if len(m.Words) != 12 {
		t.Errorf("Expected 12 words, got %d", len(m.Words))
	}
}

func TestNew(t *testing.T) {
	ent, _ := entropy.Random(128)
	m, err := New(ent, English)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if len(m.Words) != 12 {
		t.Errorf("Expected 12 words, got %d", len(m.Words))
	}
}

func TestSentence(t *testing.T) {
	m := &Mnemonic{
		Words:    []string{"hello", "world"},
		Language: English,
	}
	want := "hello world"
	got := m.Sentence()
	if got != want {
		t.Errorf("Sentence() = %q; want %q", got, want)
	}
}

func TestSentence_Japanese(t *testing.T) {
	m := &Mnemonic{
		Words:    []string{"こんにちは", "世界"},
		Language: Japanese,
	}
	want := "こんにちは　世界"
	got := m.Sentence()
	if got != want {
		t.Errorf("Sentence() = %q; want %q", got, want)
	}
}

func TestGenerateSeed(t *testing.T) {
	m, _ := NewRandom(128, English)
	seed := m.GenerateSeed("password")
	if len(seed.String()) == 0 {
		t.Error("Expected non-empty seed")
	}
}
