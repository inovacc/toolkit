package image

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

type Anchor int

// Anchor point positions.
const (
	Center Anchor = iota
	TopLeft
	Top
	TopRight
	Left
	Right
	BottomLeft
	Bottom
	BottomRight
)

type ResampleFilter struct {
	Support float64
	Kernel  func(float64) float64
}

// Lanczos filter (3 lobes).
var Lanczos ResampleFilter

var (
	ColorRed       = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	ColorGreen     = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	ColorBlue      = color.RGBA{R: 0, G: 0, B: 255, A: 255}
	ColorYellow    = color.RGBA{R: 255, G: 255, B: 0, A: 255}
	ColorCyan      = color.RGBA{R: 0, G: 255, B: 255, A: 255}
	ColorMagenta   = color.RGBA{R: 255, G: 0, B: 255, A: 255}
	ColorOrange    = color.RGBA{R: 255, G: 165, B: 0, A: 255}
	ColorPurple    = color.RGBA{R: 128, G: 0, B: 128, A: 255}
	ColorGray      = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	ColorBlack     = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	ColorWhite     = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	ColorPink      = color.RGBA{R: 255, G: 192, B: 203, A: 255}
	ColorBrown     = color.RGBA{R: 165, G: 42, B: 42, A: 255}
	ColorLime      = color.RGBA{R: 50, G: 205, B: 50, A: 255}
	ColorTeal      = color.RGBA{R: 0, G: 128, B: 128, A: 255}
	ColorNavy      = color.RGBA{R: 0, G: 0, B: 128, A: 255}
	ColorOlive     = color.RGBA{R: 128, G: 128, B: 0, A: 255}
	ColorMaroon    = color.RGBA{R: 128, G: 0, B: 0, A: 255}
	ColorLightGray = color.RGBA{R: 211, G: 211, B: 211, A: 255}
	ColorDarkGray  = color.RGBA{R: 64, G: 64, B: 64, A: 255}
)

// Error messages
var (
	ErrInvalidDimensions = errors.New("invalid image dimensions")
	ErrInvalidBrightness = errors.New("invalid brightness value")
	ErrInvalidThickness  = errors.New("invalid line thickness")
	ErrInvalidSize       = errors.New("invalid size")
	ErrInvalidAngle      = errors.New("invalid angle or skew")
	ErrUnsupportedFormat = errors.New("unsupported image format")
	ErrFileCreation      = errors.New("failed to create output file")
	ErrImageEncoding     = errors.New("failed to encode image")
	ErrImageProcessing   = errors.New("image processing failed")
)

// Format represents supported image formats
type Format string

func (f Format) String() string {
	return string(f)
}

const (
	FormatJPEG Format = "jpg"
	FormatJPG  Format = "jpeg"
	FormatPNG  Format = "png"
	FormatGIF  Format = "gif"
)

type Imager interface {
	Resize(width, height int, filter ResampleFilter) (*Image, error)
	Thumbnail(width, height int, filter ResampleFilter) (*Image, error)
	Fit(width, height int, filter ResampleFilter) (*Image, error)
	FlipH() (*Image, error)
	FlipV() (*Image, error)
	Transpose() (*Image, error)
	Transverse() (*Image, error)
	Grayscale() (*Image, error)
	Invert() (*Image, error)
	Sepia() (*Image, error)
	AdjustContrast(percentage float64) (*Image, error)
	AdjustBrightness(percentage float64) (*Image, error)
	AdjustSaturation(percentage float64) (*Image, error)
	AdjustGamma(gamma float64) (*Image, error)
	Sharpen(sigma float64) (*Image, error)
	Blur(sigma float64) (*Image, error)
	Rotate(angle float64, bgColor color.Color) (*Image, error)
	Rotate90() (*Image, error)
	Rotate180() (*Image, error)
	Rotate270() (*Image, error)
	Zoom(factor float64, anchor Anchor) (*Image, error)
	Crop(x, y, w, h int) (*Image, error)
	CropCenter(w, h int) (*Image, error)
	Fill(width int, height int, anchor Anchor, filter ResampleFilter) (*Image, error)
	DrawRect(x, y, w, h int, col color.Color, thickness int) (*Image, error)
	DrawText(x, y int, text string, fontPath string, size float64, col color.Color) (*Image, error)
	Skew(xDeg, yDeg float64) (*Image, error)
	Shear(xFactor, yFactor float64) (*Image, error)
	FromBytes(data []byte) error
	ToBase64() (string, error)
	FromBase64(data string) error
	Clone() *Image
	GetFormat() Format
	GetSize() (width, height int)
	GetAspectRatio() float64
	GetMetadata() map[string]string
	Bytes() ([]byte, error)
	ToJPEG(path string) error
	ToPNG(path string) error
	ToGIF(path string) error
	GetObject() image.Image
	Compare(other Imager, threshold uint32) ([]PixelDiff, error)
	DrawRectangles(rects []*Rectangle) error
	DrawRectangle(rect *Rectangle) error
	ToRGBA() (*image.RGBA, image.Rectangle)
}

// Image represents an image with its properties
type Image struct {
	imagePath string
	format    Format
	img       image.Image
}

// NewImage creates a new Image instance from the given path
func NewImage(imagePath string) (Imager, error) {
	fullPath, err := filepath.Abs(imagePath)
	if err != nil {
		return nil, err
	}
	imgObj := &Image{imagePath: fullPath}
	if err := imgObj.load(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImageProcessing, err)
	}
	return imgObj, nil
}

func (i *Image) GetObject() image.Image {
	return i.img
}

func (i *Image) Clone() *Image {
	return &Image{
		img:       i.img,
		imagePath: i.imagePath,
		format:    i.format,
	}
}

type PixelDiff struct {
	X, Y   int
	PixelA color.Color
	PixelB color.Color
}

func (i *Image) Compare(value Imager, threshold uint32) ([]PixelDiff, error) {
	boundsA := value.GetObject().Bounds()
	boundsB := i.img.Bounds()

	if !boundsA.Eq(boundsB) {
		return nil, fmt.Errorf("image dimensions do not match: A=%v, B=%v", boundsA, boundsB)
	}

	var diffs []PixelDiff
	for y := boundsA.Min.Y; y < boundsA.Max.Y; y++ {
		for x := boundsA.Min.X; x < boundsA.Max.X; x++ {
			cA := value.GetObject().At(x, y)
			cB := i.img.At(x, y)
			if !i.colorsEqual(cA, cB, threshold) {
				diffs = append(diffs, PixelDiff{
					X: x, Y: y,
					PixelA: cA,
					PixelB: cB,
				})
			}
		}
	}
	return diffs, nil
}

func (i *Image) Resize(width, height int, filter ResampleFilter) (*Image, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("%w: width=%d, height=%d", ErrInvalidDimensions, width, height)
	}

	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Resize(img, width, height, imaging.ResampleFilter(filter)), nil
	})
}

func (i *Image) Thumbnail(width, height int, filter ResampleFilter) (*Image, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("%w: width=%d, height=%d", ErrInvalidDimensions, width, height)
	}

	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Thumbnail(img, width, height, imaging.ResampleFilter(filter)), nil
	})
}

func (i *Image) Fit(width, height int, filter ResampleFilter) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Fit(img, width, height, imaging.ResampleFilter(filter)), nil
	})
}

func (i *Image) FlipH() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.FlipH(img), nil
	})
}

func (i *Image) FlipV() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.FlipV(img), nil
	})
}

func (i *Image) Grayscale() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Grayscale(img), nil
	})
}

func (i *Image) Rotate90() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Rotate90(img), nil
	})
}

func (i *Image) Rotate180() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Rotate180(img), nil
	})
}

func (i *Image) Rotate270() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Rotate270(img), nil
	})
}

func (i *Image) Sharpen(sigma float64) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Sharpen(img, sigma), nil
	})
}

func (i *Image) Blur(sigma float64) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Blur(img, sigma), nil
	})
}

func (i *Image) ToJPEG(imagePath string) error {
	return i.saveToFile(imagePath, func(w io.Writer) error {
		if err := jpeg.Encode(w, i.img, nil); err != nil {
			return fmt.Errorf("%w: JPEG encoding failed: %v", ErrImageEncoding, err)
		}
		return nil
	})
}

func (i *Image) ToPNG(imagePath string) error {
	return i.saveToFile(imagePath, func(w io.Writer) error {
		if err := png.Encode(w, i.img); err != nil {
			return fmt.Errorf("%w: PNG encoding failed: %v", ErrImageEncoding, err)
		}
		return nil
	})
}

func (i *Image) ToGIF(imagePath string) error {
	return i.saveToFile(imagePath, func(w io.Writer) error {
		bounds := i.img.Bounds()
		palette := i.createWebSafePalette()
		paletted := image.NewPaletted(bounds, palette)

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				paletted.Set(x, y, i.img.At(x, y))
			}
		}

		if err := gif.Encode(w, paletted, nil); err != nil {
			return fmt.Errorf("%w: GIF encoding failed: %v", ErrImageEncoding, err)
		}
		return nil
	})
}

func (i *Image) GetFormat() Format {
	return i.format
}

func (i *Image) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := i.encode(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i *Image) AdjustContrast(percentage float64) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.AdjustContrast(img, percentage), nil
	})
}

func (i *Image) Fill(width int, height int, anchor Anchor, filter ResampleFilter) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Fill(img, width, height, imaging.Anchor(anchor), imaging.ResampleFilter(filter)), nil
	})
}

func (i *Image) DrawRect(x, y, w, h int, col color.Color, thickness int) (*Image, error) {
	if thickness < 0 {
		return nil, fmt.Errorf("%w: thickness=%d", ErrInvalidThickness, thickness)
	}

	return i.transformImage(func(img image.Image) (image.Image, error) {
		dc := gg.NewContextForImage(img)
		dc.SetColor(col)
		dc.SetLineWidth(float64(thickness))
		dc.DrawRectangle(float64(x), float64(y), float64(w), float64(h))
		dc.Stroke()
		return dc.Image(), nil
	})
}

func (i *Image) DrawText(x, y int, text string, fontPath string, size float64, col color.Color) (*Image, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: size=%f", ErrInvalidSize, size)
	}

	return i.transformImage(func(img image.Image) (image.Image, error) {
		dc := gg.NewContextForImage(i.img)
		if err := dc.LoadFontFace(fontPath, size); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrImageProcessing, err)
		}
		dc.SetColor(col)
		dc.DrawStringAnchored(text, float64(x), float64(y), 0, 1)
		return dc.Image(), nil
	})
}

func (i *Image) Transpose() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		bounds := img.Bounds()
		dst := imaging.New(bounds.Dy(), bounds.Dx(), color.Transparent)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				dst.Set(y, x, img.At(x, y))
			}
		}
		return dst, nil
	})
}

func (i *Image) Transverse() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		bounds := img.Bounds()
		dst := imaging.New(bounds.Dy(), bounds.Dx(), color.Transparent)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				dst.Set(bounds.Dy()-y-1, bounds.Dx()-x-1, img.At(x, y))
			}
		}
		return dst, nil
	})
}

func (i *Image) Invert() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Invert(img), nil
	})
}

func (i *Image) Sepia() (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.AdjustSaturation(imaging.Grayscale(img), 20), nil
	})
}

func (i *Image) AdjustBrightness(percentage float64) (*Image, error) {
	if percentage < -1.0 || percentage > 1.0 {
		return nil, fmt.Errorf("%w: %f", ErrInvalidBrightness, percentage)
	}

	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.AdjustBrightness(img, percentage), nil
	})
}

func (i *Image) AdjustSaturation(percentage float64) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.AdjustSaturation(img, percentage), nil
	})
}

func (i *Image) AdjustGamma(gamma float64) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.AdjustGamma(img, gamma), nil
	})
}

func (i *Image) Rotate(angle float64, bgColor color.Color) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Rotate(img, angle, bgColor), nil
	})
}

func (i *Image) Zoom(factor float64, anchor Anchor) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		origBounds := img.Bounds()
		newW := int(float64(origBounds.Dx()) * factor)
		newH := int(float64(origBounds.Dy()) * factor)

		resized := imaging.Resize(img, newW, newH, imaging.Lanczos)

		canvas := imaging.New(origBounds.Dx(), origBounds.Dy(), color.Transparent)
		offsetX, offsetY := i.getAnchorOffset(anchor, origBounds.Dx(), origBounds.Dy(), newW, newH)

		return imaging.Paste(canvas, resized, image.Pt(offsetX, offsetY)), nil
	})
}

func (i *Image) Crop(x, y, w, h int) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.Crop(img, image.Rect(x, y, x+w, y+h)), nil
	})
}

func (i *Image) CropCenter(w, h int) (*Image, error) {
	return i.transformImage(func(img image.Image) (image.Image, error) {
		return imaging.CropCenter(img, w, h), nil
	})
}

func (i *Image) Skew(xDeg, yDeg float64) (*Image, error) {
	if math.Abs(xDeg) > 89 || math.Abs(yDeg) > 89 {
		return nil, fmt.Errorf("%w: xDeg=%.2f, yDeg=%.2f", ErrInvalidAngle, xDeg, yDeg)
	}

	xRad := xDeg * math.Pi / 180
	yRad := yDeg * math.Pi / 180

	xFactor := math.Tan(xRad)
	yFactor := math.Tan(yRad)

	return i.Shear(xFactor, yFactor)
}

func (i *Image) Shear(xFactor, yFactor float64) (*Image, error) {
	if math.Abs(xFactor) > 10 || math.Abs(yFactor) > 10 {
		return nil, fmt.Errorf("%w: xFactor=%.4f, yFactor=%.4f", ErrInvalidAngle, xFactor, yFactor)
	}

	return i.transformImage(func(img image.Image) (image.Image, error) {
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Calculate new dimensions after shearing
		newWidth := width + int(math.Abs(yFactor)*float64(height))
		newHeight := height + int(math.Abs(xFactor)*float64(width))

		// Create a new image with calculated dimensions
		dst := imaging.New(newWidth, newHeight, color.Transparent)

		// Apply shear transformation
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// Calculate a new position
				newX := x + int(float64(y)*yFactor)
				newY := y + int(float64(x)*xFactor)

				// Only set a pixel if it's within bounds
				if newX >= 0 && newX < newWidth && newY >= 0 && newY < newHeight {
					dst.Set(newX, newY, img.At(x+bounds.Min.X, y+bounds.Min.Y))
				}
			}
		}

		// Perform bilinear interpolation to fill gaps
		for y := 0; y < newHeight; y++ {
			for x := 0; x < newWidth; x++ {
				if dst.At(x, y) == color.Transparent {
					// Find nearest non-transparent pixels
					var (
						sumR, sumG, sumB, sumA float64
						count                  float64
					)

					// Check surrounding pixels
					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							nx, ny := x+dx, y+dy
							if nx >= 0 && nx < newWidth && ny >= 0 && ny < newHeight {
								c := dst.At(nx, ny)
								r, g, b, a := c.RGBA()
								if a > 0 {
									sumR += float64(r >> 8)
									sumG += float64(g >> 8)
									sumB += float64(b >> 8)
									sumA += float64(a >> 8)
									count++
								}
							}
						}
					}

					// If we found nearby pixels, interpolate
					if count > 0 {
						dst.Set(x, y, color.RGBA{
							R: uint8(sumR / count),
							G: uint8(sumG / count),
							B: uint8(sumB / count),
							A: uint8(sumA / count),
						})
					}
				}
			}
		}

		return dst, nil
	})
}

func (i *Image) GetSize() (width, height int) {
	b := i.img.Bounds()
	return b.Dx(), b.Dy()
}

func (i *Image) GetAspectRatio() float64 {
	w, h := i.GetSize()
	if h == 0 {
		return 0
	}
	return float64(w) / float64(h)
}

func (i *Image) GetMetadata() map[string]string {
	return map[string]string{
		"format": string(i.format),
		"path":   i.imagePath,
	}
}

func (i *Image) FromBytes(data []byte) error {
	i.imagePath = "memory"
	return i.decodeFromReader(bytes.NewReader(data))
}

func (i *Image) ToBase64() (string, error) {
	buf := new(bytes.Buffer)
	if err := i.encode(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func (i *Image) FromBase64(data string) error {
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrImageProcessing, err)
	}
	return i.FromBytes(b)
}

func (i *Image) ToRGBA() (*image.RGBA, image.Rectangle) {
	bounds := i.img.Bounds()
	return image.NewRGBA(bounds), bounds
}

type Rectangle struct {
	X, Y, W, H int
	Color      color.RGBA
	Thickness  int
}

func NewRectangle(x, y, w, h int, color color.RGBA, thickness int) *Rectangle {
	if thickness < 0 {
		thickness = 1
	}

	return &Rectangle{
		X:         x,
		Y:         y,
		W:         w,
		H:         h,
		Color:     color,
		Thickness: thickness,
	}
}

func (i *Image) DrawRectangles(rects []*Rectangle) error {
	for _, r := range rects {
		if err := i.DrawRectangle(r); err != nil {
			return err
		}
	}
	return nil
}

func (i *Image) DrawRectangle(r *Rectangle) error {
	rgba, bounds := i.ToRGBA()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, i.img.At(x, y))
		}
	}

	// Draw rectangle borders
	rect := image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)

	for t := 0; t < r.Thickness; t++ {
		for x := rect.Min.X; x <= rect.Max.X; x++ {
			if i.inBounds(x, rect.Min.Y+t, bounds) {
				rgba.Set(x, rect.Min.Y+t, r.Color) // Top
			}
			if i.inBounds(x, rect.Max.Y-t, bounds) {
				rgba.Set(x, rect.Max.Y-t, r.Color) // Bottom
			}
		}

		for y := rect.Min.Y; y <= rect.Max.Y; y++ {
			if i.inBounds(rect.Min.X+t, y, bounds) {
				rgba.Set(rect.Min.X+t, y, r.Color) // Left
			}
			if i.inBounds(rect.Max.X-t, y, bounds) {
				rgba.Set(rect.Max.X-t, y, r.Color) // Right
			}
		}
	}

	i.img = rgba
	return nil
}

func (i *Image) inBounds(x, y int, b image.Rectangle) bool {
	return x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y
}

func (i *Image) createWebSafePalette() color.Palette {
	palette := make(color.Palette, 0, 256)
	palette = append(palette, color.Transparent)

	for r := 0; r <= 255; r += 51 {
		for g := 0; g <= 255; g += 51 {
			for b := 0; b <= 255; b += 51 {
				if len(palette) < 256 {
					palette = append(palette, color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255})
				}
			}
		}
	}
	return palette
}

func (i *Image) encode(w io.Writer) error {
	switch i.format {
	case FormatJPEG, FormatJPG:
		return jpeg.Encode(w, i.img, nil)
	case FormatGIF:
		return gif.Encode(w, i.img, nil)
	case FormatPNG:
		return png.Encode(w, i.img)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, i.format)
	}
}

func (i *Image) decode(file *os.File, format Format) error {
	var err error
	switch format {
	case FormatJPEG, FormatJPG:
		i.img, err = jpeg.Decode(file)
	case FormatPNG:
		i.img, err = png.Decode(file)
	case FormatGIF:
		i.img, err = gif.Decode(file)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	return err
}

func (i *Image) load() error {
	imgFile, err := os.Open(i.imagePath)
	if err != nil {
		return fmt.Errorf("%w: opening file: %v", ErrImageProcessing, err)
	}

	if err := i.decodeFromReader(imgFile); err != nil {
		return fmt.Errorf("%w: decoding image: %v", ErrImageProcessing, err)
	}

	if err := imgFile.Close(); err != nil {
		return fmt.Errorf("image loaded but failed to close file: %w", err)
	}

	return nil
}

func (i *Image) getFormat(filename string) (Format, error) {
	ext := Format(strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), "."))
	switch ext {
	case FormatJPEG, FormatJPG, FormatPNG, FormatGIF:
		return ext, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
}

func (i *Image) transformImage(transform func(image.Image) (image.Image, error)) (*Image, error) {
	v, err := transform(i.img)
	return &Image{
		img:       v,
		imagePath: i.imagePath,
		format:    i.format,
	}, err
}

func (i *Image) saveToFile(path string, encoder func(io.Writer) error) error {
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFileCreation, err)
	}

	if err := encoder(out); err != nil {
		_ = out.Close()
		return err
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("image encoded but failed to close file: %w", err)
	}

	return nil
}

func (i *Image) decodeFromReader(r io.Reader) error {
	img, formatName, err := image.Decode(r)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrImageProcessing, err)
	}
	i.img = img
	i.format = Format(strings.ToLower(formatName))
	return nil
}

func (i *Image) getAnchorOffset(anchor Anchor, canvasW, canvasH, imgW, imgH int) (int, int) {
	switch anchor {
	case TopLeft:
		return 0, 0
	case Top:
		return (canvasW - imgW) / 2, 0
	case TopRight:
		return canvasW - imgW, 0
	case Left:
		return 0, (canvasH - imgH) / 2
	case Center:
		return (canvasW - imgW) / 2, (canvasH - imgH) / 2
	case Right:
		return canvasW - imgW, (canvasH - imgH) / 2
	case BottomLeft:
		return 0, canvasH - imgH
	case Bottom:
		return (canvasW - imgW) / 2, canvasH - imgH
	case BottomRight:
		return canvasW - imgW, canvasH - imgH
	default:
		return 0, 0
	}
}

func (i *Image) colorsEqual(a, b color.Color, threshold uint32) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()

	return i.absDiff(ar, br) < threshold && i.absDiff(ag, bg) < threshold && i.absDiff(ab, bb) < threshold && i.absDiff(aa, ba) < threshold
}

func (i *Image) absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
