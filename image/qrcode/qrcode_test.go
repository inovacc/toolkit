package qrcode

import (
	"testing"

	"github.com/divan/txqr/qr"
	"github.com/inovacc/toolkit/data/algorithm/random"
)

func TestNewQrcode(t *testing.T) {
	data := random.RandomString(500)

	qrcode := NewQrcode()
	if err := qrcode.Generate(data); err != nil {
		t.Fatal(err)
	}

	if err := qrcode.WriteFile(800, "qr.png"); err != nil {
		t.Fatal(err)
	}
}

func TestNewTxQrcode(t *testing.T) {
	data := random.RandomString(1000)

	qrcode := NewTxQrcode()
	if err := qrcode.Generate(data, 500, 2, 200, qr.Medium); err != nil {
		t.Fatal(err)
	}

	if err := qrcode.ToGIF("qr.gif"); err != nil {
		t.Fatal(err)
	}
}
