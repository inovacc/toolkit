package qrcode

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"image"
	"image/gif"
	"os"

	"github.com/divan/txqr"
	"github.com/divan/txqr/qr"
)

// https://divan.dev/posts/animatedqr/
// https://github.com/divan/txqr
// https://github.com/makiuchi-d/gozxing
// https://blog.devgenius.io/a-simple-qr-code-reader-service-in-golang-15483fbe55e4

type TxQrcode struct {
	data []byte
}

func NewTxQrcode() *TxQrcode {
	return &TxQrcode{}
}

func (t *TxQrcode) GenerateFrames(frames []string, imgSize, fps, size int, lvl qr.RecoveryLevel) error {
	out := &gif.GIF{
		Image: make([]*image.Paletted, len(frames)),
		Delay: make([]int, len(frames)),
	}

	for i, hexData := range frames {
		qrImg, err := qr.Encode(hexData, imgSize, lvl)
		if err != nil {
			return fmt.Errorf("QR encode: %v", err)
		}
		out.Image[i] = qrImg.(*image.Paletted)
		out.Delay[i] = t.fpsToGifDelay(fps)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		return fmt.Errorf("gif create: %v", err)
	}

	t.data = buf.Bytes()
	return nil
}

func (t *TxQrcode) Generate(data string, imgSize int, fps, size int, lvl qr.RecoveryLevel) error {
	chunks, err := txqr.NewEncoder(size).Encode(hex.EncodeToString([]byte(data)))
	if err != nil {
		return fmt.Errorf("encode: %v", err)
	}

	out := &gif.GIF{
		Image: make([]*image.Paletted, len(chunks)),
		Delay: make([]int, len(chunks)),
	}

	for i, chunk := range chunks {
		encode, err := qr.Encode(chunk, imgSize, lvl)
		if err != nil {
			return fmt.Errorf("QR encode: %v", err)
		}

		out.Image[i] = encode.(*image.Paletted)
		out.Delay[i] = t.fpsToGifDelay(fps)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		return fmt.Errorf("gif create: %v", err)
	}

	t.data = buf.Bytes()
	return nil
}

func (t *TxQrcode) ToGIF(filename string) error {
	return os.WriteFile(filename, t.data, 0660)
}

func (t *TxQrcode) fpsToGifDelay(fps int) int {
	if fps == 0 {
		return 100
	}
	return 100 / fps
}
