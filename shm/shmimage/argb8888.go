package shmimage

import (
	"image"
	"image/color"
	"image/draw"

	"deedles.dev/wl/internal/bin"
)

// ARGB8888 is an in-memory image whose At method returns color.ARGB8888 values.
type ARGB8888 struct {
	// Pix holds the image's pixels, in R, G, B, A order. The pixel at
	// (x, y) starts at Pix[(y-Rect.Min.Y)*Stride + (x-Rect.Min.X)*4].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

// NewARGB8888 returns a new ARGB8888 image with the given bounds.
func NewARGB8888(r image.Rectangle) *ARGB8888 {
	return &ARGB8888{
		Pix:    make([]uint8, r.Dx()*r.Dy()*4),
		Stride: 4 * r.Dx(),
		Rect:   r,
	}
}

func (p *ARGB8888) Bounds() image.Rectangle { return p.Rect }

func (p *ARGB8888) ColorModel() color.Model { return ARGB8888Model }

func (p *ARGB8888) At(x, y int) color.Color {
	return p.ARGB8888At(x, y)
}

func (p *ARGB8888) ARGB8888At(x, y int) ARGB8888Color {
	if !(image.Point{x, y}.In(p.Rect)) {
		return ARGB8888Color(0)
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+4 : i+4] // Small cap improves performance, see https://golang.org/issue/27857
	return bin.Value[ARGB8888Color](*(*[4]byte)(s))
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (p *ARGB8888) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*4
}

func (p *ARGB8888) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(p.Rect)) {
		return
	}
	i := p.PixOffset(x, y)
	c1 := ARGB8888Model.Convert(c).(ARGB8888Color)
	ca := bin.Bytes(c1)
	s := p.Pix[i : i+4 : i+4] // Small cap improves performance, see https://golang.org/issue/27857
	copy(s, ca[:])
}

// SubImage returns an image representing the portion of the image p visible
// through r. The returned value shares pixels with the original image.
func (p *ARGB8888) SubImage(r image.Rectangle) draw.Image {
	r = r.Intersect(p.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &ARGB8888{}
	}
	i := p.PixOffset(r.Min.X, r.Min.Y)
	return &ARGB8888{
		Pix:    p.Pix[i:],
		Stride: p.Stride,
		Rect:   r,
	}
}
