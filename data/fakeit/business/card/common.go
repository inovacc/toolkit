package card

import (
	"math/rand"
	"strconv"
	"strings"
)

type Card struct {
	Number         string
	CardholderName string
	ExpiryMonth    int
	ExpiryYear     int
	CVV            string
	Brand          string
	IssueDate      string
}

type Details struct {
	Prefix string
	Length int
}

var creditCardSpecs = map[string][]Details{
	"Visa": {{Prefix: "4", Length: 16}},
	"Mastercard": {
		{Prefix: "51", Length: 16},
		{Prefix: "52", Length: 16},
		{Prefix: "53", Length: 16},
		{Prefix: "54", Length: 16},
		{Prefix: "55", Length: 16},
	},
	"American Express": {
		{Prefix: "34", Length: 15},
		{Prefix: "37", Length: 15},
	},
	"Discover": {
		{Prefix: "6011", Length: 16},
		{Prefix: "65", Length: 16},
	},
}

var debitCardSpecs = map[string][]Details{
	"Visa Electron": {
		{Prefix: "4026", Length: 16},
		{Prefix: "417500", Length: 16},
		{Prefix: "4508", Length: 16},
		{Prefix: "4844", Length: 16},
		{Prefix: "4913", Length: 16},
		{Prefix: "4917", Length: 16},
	},
	"Maestro": {
		{Prefix: "5018", Length: 16},
		{Prefix: "5020", Length: 16},
		{Prefix: "5038", Length: 16},
		{Prefix: "6304", Length: 16},
		{Prefix: "6759", Length: 16},
		{Prefix: "6761", Length: 16},
		{Prefix: "6763", Length: 16},
	},
	"Visa Debit": {{Prefix: "4", Length: 16}},
	"Mastercard Debit": {
		{Prefix: "51", Length: 16},
		{Prefix: "52", Length: 16},
		{Prefix: "53", Length: 16},
		{Prefix: "54", Length: 16},
		{Prefix: "55", Length: 16},
	},
}

func generateNumber(prefix string, length int) string {
	remainingLength := length - len(prefix) - 1
	number := prefix
	for i := 0; i < remainingLength; i++ {
		number += strconv.Itoa(rand.Intn(10))
	}
	checkDigit := generateLuhnCheckDigit(number)
	return number + strconv.Itoa(checkDigit)
}

func generateLuhnCheckDigit(partial string) int {
	sum := 0
	double := len(partial)%2 == 0

	for i := 0; i < len(partial); i++ {
		digit := int(partial[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}

	return (10 - (sum % 10)) % 10
}

func generateCVV(length int) string {
	var cvv strings.Builder
	for i := 0; i < length; i++ {
		cvv.WriteString(strconv.Itoa(rand.Intn(10)))
	}
	return cvv.String()
}

func formatNumber(number string) string {
	var parts []string
	for i := 0; i < len(number); i += 4 {
		end := i + 4
		if end > len(number) {
			end = len(number)
		}
		parts = append(parts, number[i:end])
	}
	return strings.Join(parts, " ")
}

func cardBrand(number string) string {
	number = strings.ReplaceAll(number, " ", "")
	switch {
	case strings.HasPrefix(number, "4"):
		return "Visa"
	case strings.HasPrefix(number, "34"), strings.HasPrefix(number, "37"):
		return "American Express"
	case strings.HasPrefix(number, "5"):
		return "Mastercard"
	case strings.HasPrefix(number, "6011"), strings.HasPrefix(number, "65"):
		return "Discover"
	case strings.HasPrefix(number, "35"):
		return "JCB"
	case strings.HasPrefix(number, "30"), strings.HasPrefix(number, "36"), strings.HasPrefix(number, "38"):
		return "Diners Club"
	default:
		return "Unknown"
	}
}
