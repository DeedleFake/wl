// Package wire defines types helpful for dealing with the Wayland
// wire protocol. It is primarly intended for usage by generated code.
package wire

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"unsafe"

	"golang.org/x/sys/unix"
)

// byteOrder is the host byte order.
var byteOrder binary.ByteOrder = binary.LittleEndian

func init() {
	n := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&n))
	if b[0] == 0 {
		byteOrder = binary.BigEndian
	}
}

// read calls binary.Read() with the host byte order.
func read(r io.Reader, v any) error {
	return binary.Read(r, byteOrder, v)
}

// write calls binary.Write() with the host byte order.
func write(w io.Writer, v any) error {
	return binary.Write(w, byteOrder, v)
}

// unixTee reads from c, but also reads out-of-band data
// simultaneously, writing it into oob.
type unixTee struct {
	c   *net.UnixConn
	oob io.Writer
}

func (t unixTee) Read(buf []byte) (int, error) {
	oob := make([]byte, unix.CmsgSpace(len(buf))) // TODO: How big should this be?
	n, oobn, _, _, err := t.c.ReadMsgUnix(buf, oob)
	_, ooberr := t.oob.Write(oob[:oobn])
	return n, errors.Join(err, ooberr)
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
