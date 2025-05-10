package file2image

// import (
// 	"bytes"
// 	"crypto/sha256"
// 	"encoding/binary"
// 	"fmt"
// 	"hash/crc32"
// 	"image/png"
// 	"io"
// 	"log"
// 	"os"
// 	"path/filepath"
// 	"strings"
//
// 	"github.com/divan/txqr"
// 	"github.com/divan/txqr/qr"
// 	"github.com/inovacc/utils/v2/encoding/encoder/gob"
// 	"github.com/inovacc/utils/v2/image/qrcode"
// 	"gocv.io/x/gocv"
// )
//
// var countStart, countFragment, countEnd int
//
// func splitFile(filename string, size int) (chan []byte, error) {
// 	file, err := os.Open(filename)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	dataChan := make(chan []byte, 1)
// 	go func() {
// 		defer func() {
// 			close(dataChan)
// 			if err := file.Close(); err != nil {
// 				log.Println("Error closing file:", err)
// 			}
// 		}()
//
// 		buf := make([]byte, size)
// 		for {
// 			n, err := file.Read(buf)
// 			if err != nil {
// 				if err != io.EOF {
// 					fmt.Println("Error reading file:", err)
// 				}
// 				break
// 			}
// 			dataChan <- buf[:n]
// 		}
// 	}()
//
// 	return dataChan, nil
// }
//
// type File2Image struct {
// 	Data chan []byte
// }
//
// func NewChunks(filename string, size int) (*File2Image, error) {
// 	dataChan, err := splitFile(filename, size)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	f := &File2Image{
// 		Data: make(chan []byte),
// 	}
//
// 	go func() {
// 		defer close(f.Data)
// 		// seen := make(map[string]bool)
//
// 		for obj := range dataChan {
// 			// chunkHash := sha256.Sum256(obj.data)
// 			// hashKey := fmt.Sprintf("%x", chunkHash)
// 			// if seen[hashKey] {
// 			// 	log.Printf("🔁 Chunk duplicado ignorado (idx %d)", obj.idx)
// 			// 	continue
// 			// }
// 			// seen[hashKey] = true
//
// 			meta := &Header{
// 				crc:   obj.crc,
// 				index: fmt.Sprintf("%06X", obj.idx),
// 				mime:  getMimeType(filename),
// 				kind:  obj.kind,
// 			}
//
// 			// Start operation
// 			if meta.kind == KindOperationStart {
// 				meta.name = filepath.Base(filename)
// 				meta.hash = hash(obj.data)
// 			}
//
// 			// Fragment and End cleanup
// 			if meta.kind != KindOperationStart {
// 				meta.name = ""
// 				meta.hash = nil
// 			}
//
// 			f.Data <- f.encode2Bytes(meta, size)
// 		}
// 	}()
//
// 	return f, nil
// }
//
// func (q *File2Image) GenerateFramesLive() error {
// 	window := gocv.NewWindow("QR Dinámico")
// 	defer window.Close()
//
// 	qr := qrcode.NewQrcode()
//
// 	log.Println("Preparando transmisión...")
//
// 	for obj := range q.Data {
// 		meta := &Header{}
// 		if err := gob.DecodeGob(obj, meta); err != nil {
// 			log.Printf("Error decoding metadata: %v", err)
// 			continue
// 		}
//
// 		switch meta.kind {
// 		case KindOperationStart:
// 			countStart++
// 		case KindOperationFragment:
// 			countFragment++
// 		case KindOperationEnd:
// 			countEnd++
// 		}
//
// 		log.Printf("Header: kind=%v, index=%s, CRC=%08X, name=%s, mime=%v",
// 			meta.kind, meta.index, meta.crc, meta.name, meta.mime)
//
// 		if err := qr.GenerateRaw(obj); err != nil {
// 			return fmt.Errorf("QR generate: %v", err)
// 		}
//
// 		buf := new(bytes.Buffer)
// 		if err := png.encode(buf, qr.Image(700)); err != nil {
// 			return fmt.Errorf("png encode: %v", err)
// 		}
//
// 		mat, err := gocv.IMDecode(buf.Bytes(), gocv.IMReadGrayScale)
// 		if err != nil {
// 			return fmt.Errorf("decode to mat: %v", err)
// 		}
//
// 		if err := window.IMShow(mat); err != nil {
// 			return err
// 		}
//
// 		if window.WaitKey(100) == 113 { // tecla 'q'
// 			mat.Close()
// 			break
// 		}
// 		mat.Close()
// 	}
//
// 	log.Println("======= RESUMEN DE FRAMES =======")
// 	log.Printf("START    : %d", countStart)
// 	log.Printf("FRAGMENT : %d", countFragment)
// 	log.Printf("END      : %d", countEnd)
// 	log.Println("================================")
//
// 	summary := fmt.Sprintf("RESUMEN DE FRAMES\nSTART:%d\nFRAGMENT:%d\nEND:%d",
// 		countStart, countFragment, countEnd)
//
// 	qrFinal := qrcode.NewQrcode()
// 	if err := qrFinal.Generate(summary); err != nil {
// 		return fmt.Errorf("QR resumen: %v", err)
// 	}
//
// 	buf := new(bytes.Buffer)
// 	if err := png.encode(buf, qrFinal.Image(700)); err != nil {
// 		return fmt.Errorf("png resumen: %v", err)
// 	}
//
// 	mat, err := gocv.IMDecode(buf.Bytes(), gocv.IMReadGrayScale)
// 	if err != nil {
// 		return fmt.Errorf("decode resumen: %v", err)
// 	}
//
// 	log.Println("Mostrando resumen final...")
// 	window.IMShow(mat)
// 	window.WaitKey(0)
// 	mat.Close()
//
// 	return nil
// }
//
// func ShowTxqrSequence(frames [][]byte) error {
// 	window := gocv.NewWindow("QR Dinámico TXQR")
// 	defer window.Close()
//
// 	encoder := txqr.NewEncoder(512) // ajusta tamaño del chunk
// 	for _, frame := range frames {
// 		hexData := fmt.Sprintf("%x", frame)
// 		qrChunks, err := encoder.encode(hexData)
// 		if err != nil {
// 			return fmt.Errorf("txqr encode: %v", err)
// 		}
//
// 		for _, segment := range qrChunks {
// 			img, err := qr.encode(segment, 500, qr.Medium)
// 			if err != nil {
// 				return fmt.Errorf("txqr encode img: %v", err)
// 			}
//
// 			var buf bytes.Buffer
// 			if err := png.encode(&buf, img); err != nil {
// 				return fmt.Errorf("png encode: %v", err)
// 			}
//
// 			mat, err := gocv.IMDecode(buf.Bytes(), gocv.IMReadGrayScale)
// 			if err != nil {
// 				return fmt.Errorf("decode to mat: %v", err)
// 			}
//
// 			window.IMShow(mat)
// 			if window.WaitKey(150) == 113 {
// 				return nil
// 			}
// 			mat.Close()
// 		}
// 	}
// 	return nil
// }
