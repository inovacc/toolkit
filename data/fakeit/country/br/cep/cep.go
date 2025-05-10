package cep

import (
	"fmt"
	"regexp"
	"strings"
)

// IsValidCEP checks if a string is a valid Brazilian postal code (CEP).
// It returns true if the input contains exactly 8 numeric digits, ignoring formatting like dots or hyphens.
func IsValidCEP(input string) bool {
	input = UnformatCEP(input)
	match, _ := regexp.MatchString(`^\d{8}$`, input)
	return match
}

// FormatCEP formats a valid unformatted CEP (8 digits) into the "#####-###" format.
// Returns an empty string if the input is not exactly 8 digits.
func FormatCEP(input string) string {
	input = UnformatCEP(input)
	if len(input) != 8 {
		return ""
	}
	return fmt.Sprintf("%s-%s", input[:5], input[5:])
}

// UnformatCEP removes hyphens, dots, and whitespace from the CEP input string.
// If the resulting string is not 8 characters long, it returns an empty string.
func UnformatCEP(cep string) string {
	clean := strings.ReplaceAll(cep, "-", "")
	clean = strings.ReplaceAll(clean, ".", "")
	clean = strings.TrimSpace(clean)

	if len(clean) != 8 {
		return ""
	}
	return clean
}
