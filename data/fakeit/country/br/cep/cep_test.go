package cep

import "testing"

func TestIsValidCEP(t *testing.T) {
	if !IsValidCEP("12345678") {
		t.Errorf("Expected true, got false")
		return
	}

	if !IsValidCEP("12345-678") {
		t.Errorf("Expected true, got false")
		return
	}

	if IsValidCEP("1234-567") {
		t.Errorf("Expected false, got true")
		return
	}

	if IsValidCEP("abcde123") {
		t.Errorf("Expected false, got true")
		return
	}

	if IsValidCEP("123456789") {
		t.Errorf("Expected false, got true")
		return
	}
}
