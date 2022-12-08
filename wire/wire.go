// Package wire defines types helpful for dealing with the Wayland
// wire protocol. It is primarly intended for usage by generated code.
package wire

import (
	"errors"
	"io"
	"math"
	"net"

	"golang.org/x/sys/unix"
)

func padding(length uint32) uint32 {
	pad := 4 - (length % (32 / 8))
	if pad == 4 {
		return 0
	}
	return pad
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

// NewID represents the Wayland new_id type when it doesn't have a
// pre-defined interface.
type NewID struct {
	Interface string
	Version   uint32
	ID        uint32
}

// Object represents a Wayland protocol object.
type Object interface {
	// ID returns the ID of the object. It returns 0 before the Object
	// is added to an object management system.
	ID() uint32

	// SetID is used by the object ID management system to tell the
	// object what its own ID should be. It should almost never be
	// called manually.
	SetID(id uint32)

	// Dispatch pertforms the operation requested by the message in the
	// buffer.
	Dispatch(msg *MessageBuffer) error

	// Delete is called by the object ID management system when an
	// object is deleted.
	Delete()
}

// DebugObject is implemented by Objects that can provide debug
// information about themselves.
type DebugObject interface {
	// MethodName returns the string name of the specified local method.
	MethodName(opcode uint16) string
}

type State interface {
	Add(Object)
	Set(uint32, Object)
	Enqueue(*MessageBuilder)
}
