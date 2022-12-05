package wire

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// MessageBuilder is a message that is under construction.
type MessageBuilder struct {
	// Sender is the object on which a method is being called.
	Sender Identifier

	// Op is the opcode of the request or event of the message.
	Op uint16

	// Method is the name of the method being called. It is included
	// purely for debugging purposes.
	Method string

	// Args is the original set of arguments passed to the function from
	// which this MessageBuilder was generated. It is included purely
	// for debugging purposes.
	Args []any

	data bytes.Buffer
	fds  []int
	err  error
}

func (mb *MessageBuilder) WriteInt(v int32) {
	if mb.err != nil {
		return
	}

	write(&mb.data, v)
}

func (mb *MessageBuilder) WriteUint(v uint32) {
	if mb.err != nil {
		return
	}

	write(&mb.data, v)
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

	write(&mb.data, v)
}

func (mb *MessageBuilder) WriteString(v string) {
	if mb.err != nil {
		return
	}

	pad := padding(uint32(len(v) + 1))
	write(&mb.data, uint32(len(v)+1))
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
	write(&mb.data, uint32(len(v)))
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

	mb.fds = append(mb.fds, fd)
}

// Build builds the message and sends it to c. The MessageBuilder
// should not be used again after this method is called.
func (mb *MessageBuilder) Build(c *net.UnixConn) error {
	defer func() {
		for _, fd := range mb.fds {
			unix.Close(fd) // TODO: Handle errors.
		}
		mb.fds = nil
	}()

	if mb.err != nil {
		return mb.err
	}

	length := uint32(8 + mb.data.Len())
	msg := bytes.NewBuffer(make([]byte, 0, length))
	write(msg, mb.Sender.ID())
	write(msg, (length<<16)|uint32(mb.Op))

	io.Copy(msg, &mb.data)
	oob := unix.UnixRights(mb.fds...)

	_, _, mb.err = c.WriteMsgUnix(msg.Bytes(), oob, nil)
	return mb.err
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

	return fmt.Sprintf("%v.%v(%v)", mb.Sender, mb.Method, strings.Join(args, ", "))
}
