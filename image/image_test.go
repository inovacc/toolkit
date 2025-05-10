package image

import (
	"os"
	"path/filepath"
	"testing"
)

const testDataDir = "testdata"

func setupTestImage(t *testing.T) Imager {
	imagePath := filepath.Join(testDataDir, "lena_original.jpg")
	img, err := NewImage(imagePath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	return img
}

func TestNewImage(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid JPEG image",
			path:    filepath.Join(testDataDir, "lena_original.jpg"),
			wantErr: false,
		},
		{
			name:    "Invalid path",
			path:    "nonexistent.jpg",
			wantErr: true,
		},
		{
			name:    "Unsupported format",
			path:    "test.bmp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewImage(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImage_ToPNG(t *testing.T) {
	img := setupTestImage(t)
	outPath := filepath.Join(testDataDir, "test_output.png")
	defer os.Remove(outPath)

	if err := img.ToPNG(outPath); err != nil {
		t.Fatalf("ToPNG() error = %v", err)
	}

	// Verify the file exists and is readable
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("ToPNG() failed to create output file")
	}
}

func TestImage_ToGIF(t *testing.T) {
	img := setupTestImage(t)
	outPath := filepath.Join(testDataDir, "test_output.gif")
	defer os.Remove(outPath)

	if err := img.ToGIF(outPath); err != nil {
		t.Fatalf("ToGIF() error = %v", err)
	}

	// Verify the file exists and is readable
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("ToGIF() failed to create output file")
	}
}

func TestImage_ImageProcessingFunctions(t *testing.T) {
	img := setupTestImage(t)
	tests := []struct {
		name     string
		function func() *Image
		wantNil  bool
	}{
		{
			name: "Grayscale",
			function: func() *Image {
				v, err := img.Grayscale()
				if err != nil {
					t.Fatalf("Grayscale() error = %v", err)
				}
				return v
			},
			wantNil: false,
		},
		{
			name: "Blur",
			function: func() *Image {
				v, err := img.Blur(1.0)
				if err != nil {
					t.Fatalf("Blur() error = %v", err)
				}
				return v
			},
			wantNil: false,
		},
		{
			name: "Sharpen",
			function: func() *Image {
				v, err := img.Sharpen(1.0)
				if err != nil {
					t.Fatalf("Sharpen() error = %v", err)
				}
				return v
			},
			wantNil: false,
		},
		{
			name: "AdjustContrast",
			function: func() *Image {
				v, err := img.AdjustContrast(10)
				if err != nil {
					t.Fatalf("AdjustContrast() error = %v", err)
				}
				return v
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			if (result == nil) != tt.wantNil {
				t.Fatalf("%s() returned nil = %v, want %v", tt.name, result == nil, tt.wantNil)
			}
		})
	}
}

func TestImage_ResizeFunctions(t *testing.T) {
	img := setupTestImage(t)
	tests := []struct {
		name     string
		function func() *Image
		wantNil  bool
	}{
		{
			name: "Resize",
			function: func() *Image {
				v, err := img.Resize(100, 100, Lanczos)
				if err != nil {
					t.Fatalf("Resize() error = %v", err)
				}
				return v
			},
			wantNil: false,
		},
		{
			name: "Fit",
			function: func() *Image {
				v, err := img.Fit(100, 100, Lanczos)
				if err != nil {
					t.Fatalf("Fit() error = %v", err)
				}
				return v
			},
			wantNil: false,
		},
		{
			name: "Fill",
			function: func() *Image {
				v, err := img.Fill(100, 100, Center, Lanczos)
				if err != nil {
					t.Fatalf("Fill() error = %v", err)
				}
				return v
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			if (result == nil) != tt.wantNil {
				t.Fatalf("%s() returned nil = %v, want %v", tt.name, result == nil, tt.wantNil)
			}
		})
	}
}

func TestDrawRectangle(t *testing.T) {
	img := setupTestImage(t)

	rect := NewRectangle(170, 77, 70, 70, ColorRed, 2)
	if err := img.DrawRectangle(rect); err != nil {
		t.Fatalf("failed to draw rectangle: %v", err)
	}

	testFile := filepath.Join(testDataDir, "lena_cv_draw_rectangle_test.jpg")
	if err := img.ToJPEG(testFile); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)
}

func TestDrawRectangles(t *testing.T) {
	img := setupTestImage(t)

	rects := []*Rectangle{
		NewRectangle(170, 77, 70, 70, ColorRed, 2),
		NewRectangle(50, 80, 150, 50, ColorGreen, 2),
		NewRectangle(100, 100, 100, 100, ColorBlue, 2),
	}

	if err := img.DrawRectangles(rects); err != nil {
		t.Fatalf("failed to draw rectangles: %v", err)
	}

	testFile := filepath.Join(testDataDir, "lena_cv_draw_rectangles_test.jpg")
	if err := img.ToJPEG(testFile); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)
}
