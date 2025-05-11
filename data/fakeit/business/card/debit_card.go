package card

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

func GenerateDebitCard(actual bool) Card {
	now := time.Now()
	if !actual {
		now = gofakeit.Date()
	}
	brands := []string{"Visa Electron", "Maestro", "Visa Debit", "Mastercard Debit"}
	brand := brands[rand.Intn(len(brands))]

	specs := debitCardSpecs[brand]
	spec := specs[rand.Intn(len(specs))]

	number := generateNumber(spec.Prefix, spec.Length)
	expiryYear := now.Year() + rand.Intn(3) + 2
	expiryMonth := rand.Intn(12) + 1

	cvv := generateCVV(3)

	person := gofakeit.Person()

	return Card{
		Number:         formatNumber(number),
		CardholderName: strings.ToUpper(fmt.Sprintf("%s %s", person.FirstName, person.LastName)),
		ExpiryMonth:    expiryMonth,
		ExpiryYear:     expiryYear,
		CVV:            cvv,
		Brand:          brand,
		IssueDate:      now.Format("01/2006"),
	}
}
