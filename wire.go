package wl

import (
	"encoding/binary"
	"io"
	"math"
	"unsafe"
)

var byteOrder binary.ByteOrder = binary.LittleEndian

func init() {
	n := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&n))
	if b[0] == 0 {
		byteOrder = binary.BigEndian
	}
}

type MsgReader struct {
	sender uint32
	op     uint16
	size   uint16

	r io.Reader
}

func NewMsgReader(r io.Reader) (*MsgReader, error) {
	mr := MsgReader{
		r: r,
	}

	sender, err := mr.Uint()
	if err != nil {
		return nil, err
	}
	mr.sender = sender

	so, err := mr.Uint()
	if err != nil {
		return nil, err
	}
	size := so >> 16
	op := so & 0xFFFF
	mr.op = uint16(op)
	mr.size = uint16(size)
	mr.r = &io.LimitedReader{R: r, N: int64(size) - 8} // TODO: golang/go#51115

	return &mr, nil
}

func (r MsgReader) Sender() uint32 {
	return r.sender
}

func (r MsgReader) Op() uint16 {
	return r.op
}

func (r MsgReader) Size() uint16 {
	return r.size
}

func (r *MsgReader) read(v any) error {
	return binary.Read(r.r, byteOrder, v)
}

func (r *MsgReader) Int() (v int32, err error) {
	err = r.read(&v)
	return v, err
}

func (r *MsgReader) Uint() (v uint32, err error) {
	err = r.read(&v)
	return v, err
}

func (r *MsgReader) Fixed() (v Fixed, err error) {
	err = r.read(&v)
	return v, err
}

func (r *MsgReader) String() (v string, err error) {
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

func (r *MsgReader) NewID() (v NewID, err error) {
	name, err := r.String()
	if err != nil {
		return v, err
	}

	version, err := r.Uint()
	if err != nil {
		return v, err
	}

	return NewID{Interface: name, Version: version}, nil
}

func (r *MsgReader) Array() {
	panic("Not implemented.")
}

// TODO: Fix this and add some tests for it. It's quite likely that
// none of this actually works.
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
	return int(uint32(f) & 0xFF)
}

func (f Fixed) Float() float64 {
	i := f.Int()
	frac := f.Frac()
	return float64(i) + math.Abs(float64(frac)*math.Exp2(-8))
}

type NewID struct {
	Interface string
	Version   uint32
}
