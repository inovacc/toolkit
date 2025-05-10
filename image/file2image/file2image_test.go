package file2image

//
// import (
// 	"fmt"
// 	"os"
// 	"testing"
//
// 	"github.com/inovacc/utils/v2/random/random"
// )
//
// func TestNewMetadata(t *testing.T) {
// 	filename := "testdata/test.jpg"
// 	metadata, err := NewChunks(filename, 1024)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	var frames [][]byte
// 	for frame := range metadata.Data {
// 		frames = append(frames, frame)
// 		os.WriteFile(fmt.Sprintf("testdata/txd-%s.bin", random.RandomString(8)), frame, 0644)
// 	}
//
// 	// if err := ShowTxqrSequence(frames); err != nil {
// 	// 	log.Fatalf("error mostrando secuencia QR: %v", err)
// 	// }
//
// 	// if err := metadata.GenerateFramesLive(); err != nil {
// 	// 	t.Fatal(err)
// 	// }
// }
