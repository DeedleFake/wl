package shmimage

import "image/color"

type ARGB8888Color uint32

func NewARGB8888Color(r, g, b, a uint8) ARGB8888Color {
	return ARGB8888Color((uint32(a) << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b))
}

func (c ARGB8888Color) RGBA() (r, g, b, a uint32) {
	a = uint32(c.a()) * 0xFFFF / 0xFF
	r = uint32(c.r()) * a / 0xFF
	g = uint32(c.g()) * a / 0xFF
	b = uint32(c.b()) * a / 0xFF
	return
}

func (c ARGB8888Color) r() uint8 {
	return uint8((c & 0x00FF0000) >> 16)
}

func (c ARGB8888Color) g() uint8 {
	return uint8((c & 0x0000FF00) >> 8)
}

func (c ARGB8888Color) b() uint8 {
	return uint8(c & 0x000000FF)
}

func (c ARGB8888Color) a() uint8 {
	return uint8((c & 0xFF000000) >> 24)
}

var ARGB8888Model color.Model = color.ModelFunc(argb8888Model)

func argb8888Model(c color.Color) color.Color {
	switch c := c.(type) {
	case ARGB8888Color:
		return c
	case color.NRGBA:
		return NewARGB8888Color(c.R, c.G, c.B, c.A)
	default:
		r, g, b, a := c.RGBA()
		r = r * 0xFF / a
		g = g * 0xFF / a
		b = b * 0xFF / a
		a = a * 0xFF / 0xFFFF
		return NewARGB8888Color(uint8(r), uint8(g), uint8(b), uint8(a))
	}
}
