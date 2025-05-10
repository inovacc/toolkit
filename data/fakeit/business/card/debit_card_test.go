package card

import (
	"fmt"
	"testing"
)

func TestGenerateDebitCard(t *testing.T) {
	// Generate a debit card
	debitCard := GenerateDebitCard(true)
	fmt.Printf("Debit Card:\n")
	fmt.Printf("Brand: %s\n", debitCard.Brand)
	fmt.Printf("Number: %s\n", debitCard.Number)
	fmt.Printf("name: %s\n", debitCard.CardholderName)
	fmt.Printf("Expiry: %02d/%d\n", debitCard.ExpiryMonth, debitCard.ExpiryYear)
	fmt.Printf("CVV: %s\n", debitCard.CVV)
	fmt.Printf("Issue Date: %v\n", debitCard.IssueDate)
}
