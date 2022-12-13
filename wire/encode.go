package wire

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"deedles.dev/wl/bin"
	"golang.org/x/sys/unix"
)

// MessageBuilder is a message that is under construction.
type MessageBuilder struct {
	// Method is the name of the method being called. It is included
	// purely for debugging purposes.
	Method string

	// Args is the original set of arguments passed to the function from
	// which this MessageBuilder was generated. It is included purely
	// for debugging purposes.
	Args []any

	sender Object
	op     uint16
	data   bytes.Buffer
	fds    []int
	err    error
}

func NewMessage(sender Object, op uint16) *MessageBuilder {
	return &MessageBuilder{
		sender: sender,
		op:     op,
	}
}

func (mb *MessageBuilder) Sender() Object {
	return mb.sender
}

func (mb *MessageBuilder) Op() uint16 {
	return mb.op
}

func (mb *MessageBuilder) WriteInt(v int32) {
	if mb.err != nil {
		return
	}

	bin.Write(&mb.data, v)
}

func (mb *MessageBuilder) WriteUint(v uint32) {
	if mb.err != nil {
		return
	}

	bin.Write(&mb.data, v)
}

func (mb *MessageBuilder) WriteObject(v Object) {
	var id uint32
	if !isNil(v) {
		id = v.ID()
	}
	mb.WriteUint(id)
}

func (mb *MessageBuilder) WriteNewID(v NewID) {
	if mb.err != nil {
		return
	}

	mb.WriteString(v.Interface)
	mb.WriteUint(v.Version)
	mb.WriteUint(v.ID)
}

func (mb *MessageBuilder) WriteFixed(v Fixed) {
	if mb.err != nil {
		return
	}

	bin.Write(&mb.data, v)
}

func (mb *MessageBuilder) WriteString(v string) {
	if mb.err != nil {
		return
	}

	pad := padding(uint32(len(v) + 1))
	bin.Write(&mb.data, uint32(len(v)+1))
	mb.data.WriteString(v)
	mb.data.WriteByte(0)
	for i := uint32(0); i < pad; i++ {
		mb.data.WriteByte(0)
	}
}

func (mb *MessageBuilder) WriteArray(v []byte) {
	if mb.err != nil {
		return
	}

	pad := padding(uint32(len(v)))
	bin.Write(&mb.data, uint32(len(v)))
	mb.data.Write(v)
	for i := uint32(0); i < pad; i++ {
		mb.data.WriteByte(0)
	}
}

func (mb *MessageBuilder) WriteFile(v *os.File) {
	fd, err := unix.Dup(int(v.Fd()))
	if err != nil {
		mb.err = err
		return
	}

	if len(mb.fds) == 0 {
		runtime.SetFinalizer(mb, (*MessageBuilder).close)
	}

	mb.fds = append(mb.fds, fd)
}

// Build builds the message and sends it to c. The MessageBuilder
// should not be used again after this method is called.
func (mb *MessageBuilder) Build(c *Conn) error {
	if mb.err != nil {
		return mb.err
	}

	length := uint32(8 + mb.data.Len())
	msg := bytes.NewBuffer(make([]byte, 0, length))
	bin.Write(msg, mb.sender.ID())
	bin.Write(msg, (length<<16)|uint32(mb.op))

	io.Copy(msg, &mb.data)
	oob := unix.UnixRights(mb.fds...)

	_, _, mb.err = c.conn.WriteMsgUnix(msg.Bytes(), oob, nil)
	return mb.err
}

func (mb *MessageBuilder) close() {
	errs := make([]error, 0, len(mb.fds))
	for _, fd := range mb.fds {
		errs = append(errs, unix.Close(fd))
	}
	if mb.err == nil {
		mb.err = errors.Join(errs...)
	}
	mb.fds = nil
	runtime.SetFinalizer(mb, nil)
}

func (mb *MessageBuilder) String() string {
	args := make([]string, 0, len(mb.Args))
	for _, arg := range mb.Args {
		switch arg := arg.(type) {
		case string:
			args = append(args, strconv.Quote(arg))
		case *os.File:
			args = append(args, fmt.Sprint(arg.Fd()))
		default:
			args = append(args, fmt.Sprint(arg))
		}
	}

	return fmt.Sprintf("%v.%v(%v)", mb.sender, mb.Method, strings.Join(args, ", "))
}

func isNil(v any) bool {
	return (v == nil) || ((*[2]uintptr)(unsafe.Pointer(&v))[1] == 0)
}
