// Package cnpj provides utilities for generating, validating, formatting,
// and handling alphanumeric CNPJ identifiers based on modulo-11 checksum rules.
// Reference: https://www.serpro.gov.br/menu/noticias/videos/calculodvcnpjalfanaumerico.pdf

package cnpj

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Character-to-numeric value mapping for alphanumeric CNPJ (0–9 and A–Z)
var charToValue = map[rune]int{
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
	'A': 17, 'B': 18, 'C': 19, 'D': 20, 'E': 21, 'F': 22, 'G': 23, 'H': 24, 'I': 25, 'J': 26,
	'K': 27, 'L': 28, 'M': 29, 'N': 30, 'O': 31, 'P': 32, 'Q': 33, 'R': 34, 'S': 35, 'T': 36,
	'U': 37, 'V': 38, 'W': 39, 'X': 40, 'Y': 41, 'Z': 42,
}

// calculateCheckDigit returns a modulo-11 check digit for a 12- or 13-character alphanumeric CNPJ string.
// Returns -1 if invalid characters are found.
func calculateCheckDigit(cnpj string) int {
	weights := []int{2, 3, 4, 5, 6, 7, 8, 9}
	sum := 0
	j := 0

	for i := len(cnpj) - 1; i >= 0; i-- {
		val, ok := charToValue[rune(cnpj[i])]
		if !ok {
			return -1
		}
		sum += val * weights[j]
		j = (j + 1) % len(weights)
	}

	remainder := sum % 11
	if remainder == 0 || remainder == 1 {
		return 0
	}
	return 11 - remainder
}

// GenerateCNPJ creates a random, valid alphanumeric CNPJ (14 characters) with checksum digits.
func GenerateCNPJ() string {
	var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder

	for i := 0; i < 12; i++ {
		if rng.Intn(2) == 0 {
			sb.WriteByte(byte('0' + rng.Intn(10)))
		} else {
			sb.WriteByte(byte('A' + rng.Intn(26)))
		}
	}

	cnpjBase := sb.String()
	dv1 := calculateCheckDigit(cnpjBase)
	dv2 := calculateCheckDigit(cnpjBase + strconv.Itoa(dv1))

	return fmt.Sprintf("%s%d%d", cnpjBase, dv1, dv2)
}

// ValidateCNPJ checks whether a given alphanumeric CNPJ is valid by verifying both check digits.
func ValidateCNPJ(cnpj string) bool {
	cnpj = UnformatCNPJ(cnpj)
	if len(cnpj) != 14 {
		return false
	}

	base := cnpj[:12]
	dv1, _ := strconv.Atoi(string(cnpj[12]))
	dv2, _ := strconv.Atoi(string(cnpj[13]))

	return calculateCheckDigit(base) == dv1 && calculateCheckDigit(base+strconv.Itoa(dv1)) == dv2
}

// FormatCNPJ applies the standard CNPJ mask "##.###.###/####-##" to a valid alphanumeric string.
func FormatCNPJ(cnpj string) string {
	cnpj = UnformatCNPJ(cnpj)
	if len(cnpj) != 14 {
		return "Invalid CNPJ"
	}

	mask := "##.###.###/####-##"
	result := make([]rune, len(mask))
	cnpjIndex := 0

	for i, r := range mask {
		if r == '#' {
			result[i] = rune(cnpj[cnpjIndex])
			cnpjIndex++
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// UnformatCNPJ removes all non-alphanumeric characters and uppercases all letters in the CNPJ.
func UnformatCNPJ(cnpj string) string {
	re := regexp.MustCompile(`[^0-9A-Z]`)
	return strings.ToUpper(re.ReplaceAllString(cnpj, ""))
}
