package qrcode

import (
	"image"
	"io"

	"github.com/skip2/go-qrcode"
)

type Qrcode struct {
	qr *qrcode.QRCode
}

func NewQrcode() *Qrcode {
	return &Qrcode{}
}

func (q *Qrcode) Generate(content string) error {
	var err error
	q.qr, err = qrcode.New(content, qrcode.Highest)
	if err != nil {
		return err
	}
	return nil
}

func (q *Qrcode) GenerateRaw(data []byte) error {
	var err error
	q.qr, err = qrcode.NewWithForcedVersion(string(data), 40, qrcode.Medium)
	if err != nil {
		return err
	}
	return nil
}

func (q *Qrcode) ToPNG(size int) ([]byte, error) {
	return q.qr.PNG(size)
}

func (q *Qrcode) Image(size int) image.Image {
	return q.qr.Image(size)
}

func (q *Qrcode) Write(size int, out io.Writer) error {
	return q.qr.Write(size, out)
}

func (q *Qrcode) WriteFile(size int, filename string) error {
	return q.qr.WriteFile(size, filename)
}
