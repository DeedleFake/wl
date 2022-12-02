package wire

import (
	"bytes"
	"io"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// MessageBuilder is a message that is under construction.
type MessageBuilder struct {
	// Sender is the object ID of the sender of the message.
	Sender uint32

	// Op is the opcode of the request or event of the message.
	Op uint16

	data bytes.Buffer
	fds  []int
}

func (mb *MessageBuilder) WriteInt(v int32) {
	write(&mb.data, v)
}

func (mb *MessageBuilder) WriteUint(v uint32) {
	write(&mb.data, v)
}

func (mb *MessageBuilder) WriteNewID(v NewID) {
	mb.WriteString(v.Interface)
	mb.WriteUint(v.Version)
	mb.WriteUint(v.ID)
}

func (mb *MessageBuilder) WriteFixed(v Fixed) {
	write(&mb.data, v)
}

func (mb *MessageBuilder) WriteString(v string) {
	write(&mb.data, uint32(len(v)+1))
	mb.data.WriteString(v)
	mb.data.WriteByte(0)
}

func (mb *MessageBuilder) WriteArray(v []byte) {
	write(&mb.data, uint32(len(v)))
	mb.data.Write(v)
}

func (mb *MessageBuilder) WriteFile(v *os.File) {
	mb.fds = append(mb.fds, int(v.Fd()))
}

// Build builds the message and sends it to c. The MessageBuilder
// should not be used again after this method is called.
func (mb *MessageBuilder) Build(c *net.UnixConn) error {
	length := uint32(8 + mb.data.Len())
	msg := bytes.NewBuffer(make([]byte, 0, length))
	write(msg, mb.Sender)
	write(msg, (length<<16)|uint32(mb.Op))

	io.Copy(msg, &mb.data)
	oob := unix.UnixRights(mb.fds...)

	_, _, err := c.WriteMsgUnix(msg.Bytes(), oob, nil)
	return err
}
