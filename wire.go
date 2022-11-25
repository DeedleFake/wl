package wl

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"unsafe"

	"golang.org/x/sys/unix"
)

var byteOrder binary.ByteOrder = binary.LittleEndian

func init() {
	n := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&n))
	if b[0] == 0 {
		byteOrder = binary.BigEndian
	}
}

func read(r io.Reader, v any) error {
	return binary.Read(r, byteOrder, v)
}

func write(w io.Writer, v any) error {
	return binary.Write(w, byteOrder, v)
}

type MsgReader struct {
	sender uint32
	op     uint16
	size   uint16
	data   bytes.Reader
	oob    bytes.Reader
}

func NewMsgReader(c *net.UnixConn) (*MsgReader, error) {
	var mr MsgReader

	var oob bytes.Buffer
	r := unixTee{c: c, oob: &oob}

	err := read(r, &mr.sender)
	if err != nil {
		return nil, err
	}

	var so uint32
	err = read(r, &so)
	if err != nil {
		return nil, err
	}
	mr.size = uint16(so >> 16)
	mr.op = uint16(so & 0xFFFF)

	data := bytes.NewBuffer(make([]byte, 0, mr.size))
	_, err = io.CopyN(data, r, int64(mr.size)-8)
	if err != nil {
		return nil, err
	}

	mr.data.Reset(data.Bytes())
	mr.oob.Reset(oob.Bytes())

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

func (r *MsgReader) Int() (v int32, err error) {
	err = read(&r.data, &v)
	return v, err
}

func (r *MsgReader) Uint() (v uint32, err error) {
	err = read(&r.data, &v)
	return v, err
}

func (r *MsgReader) Fixed() (v Fixed, err error) {
	err = read(&r.data, &v)
	return v, err
}

func (r *MsgReader) String() (v string, err error) {
	length, err := r.Uint()
	if err != nil {
		return v, err
	}
	pad := length % (32 / 8)

	buf := make([]byte, length+pad)
	_, err = io.ReadFull(&r.data, buf)
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

func (r *MsgReader) Fd() (int, error) {
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
