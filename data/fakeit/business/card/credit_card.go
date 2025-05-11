package card

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

func GenerateCreditCard(actual bool) Card {
	now := time.Now()
	if !actual {
		now = gofakeit.Date()
	}
	brands := []string{"Visa", "Mastercard", "American Express", "Discover"}
	brand := brands[rand.Intn(len(brands))]

	specs := creditCardSpecs[brand]
	spec := specs[rand.Intn(len(specs))]

	number := generateNumber(spec.Prefix, spec.Length)
	expiryYear := now.Year() + rand.Intn(3) + 3
	expiryMonth := rand.Intn(12) + 1

	cvvLength := 3
	if brand == "American Express" {
		cvvLength = 4
	}
	cvv := generateCVV(cvvLength)

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
