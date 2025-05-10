package structure

import (
	"reflect"
	"testing"
)

func TestBlankStructure(t *testing.T) {
	p1 := gofakeit.Person()

	if err := BlankStructure(p1); err != nil {
		t.Error(err)
	}

	p2 := &gofakeit.PersonInfo{
		FirstName: "string",
		LastName:  "string",
		Gender:    "string",
		SSN:       "string",
		Hobby:     "string",
		Job: &gofakeit.JobInfo{
			Company:    "string",
			Title:      "string",
			Descriptor: "string",
			Level:      "string",
		},
		Address: &gofakeit.AddressInfo{
			Address:   "string",
			Street:    "string",
			City:      "string",
			State:     "string",
			Zip:       "string",
			Country:   "string",
			Latitude:  0,
			Longitude: 0,
		},
		Contact: &gofakeit.ContactInfo{
			Phone: "string",
			Email: "string",
		},
		CreditCard: &gofakeit.CreditCardInfo{
			Type:   "string",
			Number: "string",
			Exp:    "string",
			Cvv:    "string",
		},
	}

	if !reflect.DeepEqual(p1, p2) {
		t.Errorf("Expected %v, got %v", p2, p1)
	}
}
