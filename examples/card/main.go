package main

import (
	"fmt"

	"github.com/inovacc/toolkit/data/fakeit/business/card"
)

func main() {
	// Generate a credit card actually
	creditCard := card.GenerateCreditCard(true)
	fmt.Printf("Credit Card:\n")
	fmt.Printf("Brand: %s\n", creditCard.Brand)
	fmt.Printf("Number: %s\n", creditCard.Number)
	fmt.Printf("name: %s\n", creditCard.CardholderName)
	fmt.Printf("Expiry: %02d/%d\n", creditCard.ExpiryMonth, creditCard.ExpiryYear)
	fmt.Printf("CVV: %s\n", creditCard.CVV)
	fmt.Printf("Issue Date: %v\n\n", creditCard.IssueDate)

	// Generate a credit card expired
	creditCard = card.GenerateCreditCard(false)
	fmt.Printf("Credit Card:\n")
	fmt.Printf("Brand: %s\n", creditCard.Brand)
	fmt.Printf("Number: %s\n", creditCard.Number)
	fmt.Printf("name: %s\n", creditCard.CardholderName)
	fmt.Printf("Expiry: %02d/%d\n", creditCard.ExpiryMonth, creditCard.ExpiryYear)
	fmt.Printf("CVV: %s\n", creditCard.CVV)
	fmt.Printf("Issue Date: %v\n\n", creditCard.IssueDate)

	// Generate a debit card
	debitCard := card.GenerateDebitCard(true)
	fmt.Printf("Debit Card:\n")
	fmt.Printf("Brand: %s\n", debitCard.Brand)
	fmt.Printf("Number: %s\n", debitCard.Number)
	fmt.Printf("name: %s\n", debitCard.CardholderName)
	fmt.Printf("Expiry: %02d/%d\n", debitCard.ExpiryMonth, debitCard.ExpiryYear)
	fmt.Printf("CVV: %s\n", debitCard.CVV)
	fmt.Printf("Issue Date: %v\n", debitCard.IssueDate)
}
