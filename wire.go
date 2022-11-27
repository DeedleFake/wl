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
	"strings"
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

// MessageBuffer holds message data that has been read from the socket
// but not yet decoded.
type MessageBuffer struct {
	sender  uint32
	op      uint16
	size    uint16
	data    bytes.Reader
	fds     []int
	fdindex int
	err     error
}

// ReadMessage reads message data from the socket into a buffer.
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

// Sender is the object ID of the sender of the message.
func (r MessageBuffer) Sender() uint32 {
	return r.sender
}

// Op is the opcode of the message.
func (r MessageBuffer) Op() uint16 {
	return r.op
}

// Size is the total size of the message, including the 8 byte header.
func (r MessageBuffer) Size() uint16 {
	return r.size
}

func (r MessageBuffer) Err() error {
	return r.err
}

func (r *MessageBuffer) ReadInt() (v int32) {
	if r.err != nil {
		return
	}

	r.err = read(&r.data, &v)
	return v
}

func (r *MessageBuffer) ReadUint() (v uint32) {
	if r.err != nil {
		return
	}

	r.err = read(&r.data, &v)
	return v
}

func (r *MessageBuffer) ReadFixed() (v Fixed) {
	if r.err != nil {
		return
	}

	r.err = read(&r.data, &v)
	return v
}

func (r *MessageBuffer) ReadString() string {
	if r.err != nil {
		return ""
	}

	length := r.ReadUint()
	if r.err != nil {
		return ""
	}
	pad := length % (32 / 8)

	var str strings.Builder
	str.Grow(int(length + pad))
	_, r.err = io.CopyN(&str, &r.data, int64(length+pad))
	if r.err != nil {
		return ""
	}
	v := str.String()
	if v[length] != 0 {
		r.err = errors.New("string is not null-terminated")
		return ""
	}

	return v[:length-1]
}

func (r *MessageBuffer) ReadArray() []byte {
	if r.err != nil {
		return nil
	}

	length := r.ReadUint()
	if r.err != nil {
		return nil
	}
	pad := length % (32 / 8)

	buf := make([]byte, length+pad)
	_, r.err = io.ReadFull(&r.data, buf)
	if r.err != nil {
		return nil
	}

	return buf[:length]
}

func (r *MessageBuffer) ReadFile() *os.File {
	if r.err != nil {
		return nil
	}

	if r.fdindex >= len(r.fds) {
		r.err = errors.New("no more file descriptors")
		return nil
	}

	f := os.NewFile(uintptr(r.fds[r.fdindex]), "")
	r.fdindex++
	return f
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
