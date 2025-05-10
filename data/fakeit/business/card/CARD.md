# Card Generator Module

This Go module provides clean and simple tools to generate random **credit** and **debit** card information.

Fully modular and production-ready.

---

## 📦 Structure

| File | Purpose |
|:----|:--------|
| `common.go` | Common types (`Card`, `CardDetails`) and card specifications |
| `utils.go` | Helper functions (Luhn check digit, CVV generator, format number, card brand detection) |
| `credit_card.go` | Logic to generate **credit cards** (Visa, Mastercard, Amex, Discover) |
| `debit_card.go` | Logic to generate **debit cards** (Maestro, Visa Electron, etc.) |


---

## 🛠️ Usage

```go
package main

import (
	"fmt"

	"github.com/inovacc/utils/v2/data/fakeit/business/card"
)

func main() {
	// Generate a credit card
	creditCard := card.GenerateCreditCard(true)
	fmt.Printf("Credit Card:\n")
	fmt.Printf("Brand: %s\n", creditCard.Brand)
	fmt.Printf("Number: %s\n", creditCard.Number)
	fmt.Printf("Name: %s\n", creditCard.CardholderName)
	fmt.Printf("Expiry: %02d/%d\n", creditCard.ExpiryMonth, creditCard.ExpiryYear)
	fmt.Printf("CVV: %s\n", creditCard.CVV)
	fmt.Printf("Issue Date: %v\n\n", creditCard.IssueDate)

	// Generate a debit card
	debitCard := card.GenerateDebitCard(true)
	fmt.Printf("Debit Card:\n")
	fmt.Printf("Brand: %s\n", debitCard.Brand)
	fmt.Printf("Number: %s\n", debitCard.Number)
	fmt.Printf("Name: %s\n", debitCard.CardholderName)
	fmt.Printf("Expiry: %02d/%d\n", debitCard.ExpiryMonth, debitCard.ExpiryYear)
	fmt.Printf("CVV: %s\n", debitCard.CVV)
	fmt.Printf("Issue Date: %v\n", debitCard.IssueDate)
}
```

---

## ✨ Features

- Random card generation following real-world brand prefixes.
- Luhn check digit validation.
- Proper CVV generation (3 or 4 digits depending on brand).
- Expiry date based on card type (longer for credit, shorter for debit).
- Automatic card number formatting.
- Simple, clean and idiomatic Go.


---

## 🔥 Example Output

```bash
Credit Card: {Number: 4556 7375 8683 4895 CardholderName:SAMPLE CARD ExpiryMonth:7 ExpiryYear:2029 CVV:721 Brand:Visa IssueDate:2025-04-26 00:00:00 +0000 UTC}
Debit Card:  {Number: 5018 1234 5678 9123 CardholderName:SAMPLE CARD ExpiryMonth:11 ExpiryYear:2027 CVV:834 Brand:Maestro IssueDate:2025-04-26 00:00:00 +0000 UTC}
```

---

## 🧠 Notes

- You can extend easily to add more card brands.
- Safe to embed into microservices or CLI tools.

---

> Inspired by **Go Proverbs**: "Clear is better than clever". 😉
