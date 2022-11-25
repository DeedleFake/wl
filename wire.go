package wl

import (
	"encoding/binary"
	"io"
	"math"
	"unsafe"
)

type Reader struct {
	// ByteOrder is the byte order used for decoding. It defaults to
	// [binary.LittleEndian].
	ByteOrder binary.ByteOrder

	r io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		ByteOrder: binary.LittleEndian,
		r:         r,
	}
}

func (r *Reader) read(v any) error {
	return binary.Read(r.r, r.ByteOrder, v)
}

func (r *Reader) Int() (v int32, err error) {
	err = r.read(&v)
	return v, err
}

func (r *Reader) Uint() (v uint32, err error) {
	err = r.read(&v)
	return v, err
}

func (r *Reader) Fixed() (v Fixed, err error) {
	err = r.read(&v)
	return v, err
}

func (r *Reader) String() (v string, err error) {
	length, err := r.Uint()
	if err != nil {
		return v, err
	}
	pad := length % (32 / 8)

	buf := make([]byte, length+pad)
	_, err = io.ReadFull(r.r, buf)
	if err != nil {
		return v, err
	}

	return unsafe.String(&buf[0], length-1), nil
}

func (r *Reader) Object() (uint32, error) {
	return r.Uint()
}

func (r *Reader) NewID() {
	panic("Not implemented.")
}

func (r *Reader) Array() {
	panic("Not implemented.")
}

type Fixed int32

func FixedInt(v int) Fixed {
	return Fixed(v << 8)
}

func FixedFloat(v float64) Fixed {
	i, frac := math.Modf(v)
	return Fixed((int(i) << 8) | int(math.Abs(frac)*math.Exp2(8)))
}

func (f Fixed) Int() int {
	return int(f >> 8)
}

func (f Fixed) Frac() int {
	return int((uint32(f) << 24) >> 24)
}

func (f Fixed) Float() float64 {
	i := f.Int()
	frac := f.Frac()
	return float64(i) + math.Abs(float64(frac)*math.Exp2(-8))
}
