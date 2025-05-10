package gob

import (
	"reflect"
	"testing"
)

func TestEncodeGob(t *testing.T) {
	d1 := struct {
		Name string
		Age  int
		Pi   float64
	}{
		"Ana",
		24,
		3.1416,
	}

	v, err := EncodeGob(d1)
	if err != nil {
		t.Error(err)
		return
	}

	if v == nil {
		t.Error("value is null")
		return
	}

	d2 := struct {
		Name string
		Age  int
		Pi   float64
	}{}

	if err := DecodeGob(v, &d2); err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(d1, d2) {
		t.Error("struct are not equal")
		return
	}
}
