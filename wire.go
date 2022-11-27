package wl

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"reflect"
	"strings"
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

type MessageBuffer struct {
	sender  uint32
	op      uint16
	size    uint16
	data    bytes.Reader
	fds     []int
	fdindex int
}

func ReadMessage(c *net.UnixConn) (*MessageBuffer, error) {
	var mr MessageBuffer

	var oob bytes.Buffer
	r := unixTee{c: c, oob: &oob}

	err := read(r, &mr.sender)
	if err != nil {
		return nil, fmt.Errorf("read message sender: %w", err)
	}

	var so uint32
	err = read(r, &so)
	if err != nil {
		return nil, fmt.Errorf("read message size and opcode: %w", err)
	}
	mr.size = uint16(so >> 16)
	mr.op = uint16(so & 0xFFFF)

	data := bytes.NewBuffer(make([]byte, 0, mr.size))
	_, err = io.CopyN(data, r, int64(mr.size)-8)
	if err != nil {
		return nil, fmt.Errorf("copy data to buffer: %w", err)
	}

	cmsgs, err := unix.ParseSocketControlMessage(oob.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parse socket control messages: %w", err)
	}
	for _, cmsg := range cmsgs {
		fds, err := unix.ParseUnixRights(&cmsg)
		if err != nil {
			if errors.Is(err, unix.EINVAL) {
				continue
			}
			return nil, fmt.Errorf("parse unix control message: %w", err)
		}
		mr.fds = append(mr.fds, fds...)
	}

	mr.data.Reset(data.Bytes())

	return &mr, nil
}

func (r MessageBuffer) Sender() uint32 {
	return r.sender
}

func (r MessageBuffer) Op() uint16 {
	return r.op
}

func (r MessageBuffer) Size() uint16 {
	return r.size
}

func Decode(buf *MessageBuffer, val any) error {
	switch val := any(val).(type) {
	case *int32, *uint32, *Fixed:
		return read(&buf.data, val)

	case *string:
		var length uint32
		err := read(&buf.data, &length)
		if err != nil {
			return err
		}
		pad := length % (32 / 8)

		var str strings.Builder
		str.Grow(int(length + pad))
		_, err = io.CopyN(&str, &buf.data, int64(length+pad))
		if err != nil {
			return err
		}
		if str.String()[length-1] != 0 {
			return errors.New("string is not null-terminated")
		}

		*val = str.String()[:length-1]
		return nil

	case *NewID:
		var inter string
		err := Decode(buf, &inter)
		if err != nil {
			return err
		}

		var version uint32
		err = read(&buf.data, &version)
		if err != nil {
			return err
		}

		*val = NewID{Interface: inter, Version: version}
		return nil

	case **os.File:
		if buf.fdindex >= len(buf.fds) {
			return errors.New("no more file descriptors")
		}

		*val = os.NewFile(uintptr(buf.fds[buf.fdindex]), "")
		buf.fdindex++
		return nil

	default:
		v := reflect.Indirect(reflect.ValueOf(val))
		switch v.Kind() {
		case reflect.Slice:
			var length uint32
			err := read(&buf.data, &length)
			if err != nil {
				return err
			}
			// Wait, why padding? All elements are already padded, so no
			// array should require any padding, should it? Is this is a
			// mistake in the documentation?
			//pad := length % (32 / (8 * size(v.Type().Elem()))

			s := reflect.MakeSlice(v.Type(), int(length), int(length))
			for i := 0; i < s.Len(); i++ {
				err = Decode(buf, s.Index(i).Addr().Interface())
				if err != nil {
					return err
				}
			}

			v.Set(s)
			return nil
		}

		panic(fmt.Errorf("unexpected type: %T", val))
	}
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
