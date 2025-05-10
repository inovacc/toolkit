package mnemonic

import "testing"

func TestRandomWord(t *testing.T) {
	word := RandomWord(English)

	if word == "" {
		t.Errorf("RandomWord() = %v; want a word", word)
	}

	if len(word) < 3 {
		t.Errorf("RandomWord() = %v; want a word with at least 3 characters", word)
	}
}

func TestGenerateMnemonic(t *testing.T) {
	mnemonic := GenerateMnemonic(12, English)

	if len(mnemonic) == 0 {
		t.Errorf("GenerateMnemonic() = %v; want a mnemonic", mnemonic)
	}
}
