package wire

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"deedles.dev/wl/internal/bin"
	"golang.org/x/sys/unix"
)

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
	args    []any
	method  string
}

// ReadMessage reads message data from the socket into a buffer.
func ReadMessage(c *net.UnixConn) (*MessageBuffer, error) {
	var mr MessageBuffer

	var oob bytes.Buffer
	r := unixTee{c: c, oob: &oob}

	sender, err := bin.Read[uint32](r)
	if err != nil {
		return nil, fmt.Errorf("read message sender: %w", err)
	}
	mr.sender = sender

	so, err := bin.Read[uint32](r)
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
	if errors.Is(r.err, io.EOF) {
		if r.data.Size() < int64(r.size)-8 {
			return io.ErrUnexpectedEOF
		}
		return nil
	}
	return r.err
}

func (r *MessageBuffer) ReadInt() (v int32) {
	if r.err != nil {
		return
	}

	v, r.err = bin.Read[int32](&r.data)
	r.args = append(r.args, v)
	return v
}

func (r *MessageBuffer) ReadUint() (v uint32) {
	if r.err != nil {
		return
	}

	v, r.err = bin.Read[uint32](&r.data)
	r.args = append(r.args, v)
	return v
}

func (r *MessageBuffer) ReadNewID() NewID {
	return NewID{
		Interface: r.ReadString(),
		Version:   r.ReadUint(),
		ID:        r.ReadUint(),
	}
}

func (r *MessageBuffer) ReadFixed() (v Fixed) {
	if r.err != nil {
		return
	}

	v, r.err = bin.Read[Fixed](&r.data)
	r.args = append(r.args, v)
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
	pad := padding(length)

	var str strings.Builder
	str.Grow(int(length + pad))
	_, r.err = io.CopyN(&str, &r.data, int64(length+pad))
	if r.err != nil {
		return ""
	}
	v := str.String()
	if v[length-1] != 0 {
		r.err = errors.New("string is not null-terminated")
		return ""
	}

	r.args = append(r.args, v[:length-1])
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
	pad := padding(length)

	buf := make([]byte, length+pad)
	_, r.err = io.ReadFull(&r.data, buf)
	if r.err != nil {
		return nil
	}

	r.args = append(r.args, buf[:length])
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
	r.args = append(r.args, f)
	return f
}

func (r *MessageBuffer) Debug(sender Object) string {
	args := make([]string, 0, len(r.args))
	for _, arg := range r.args {
		switch arg := arg.(type) {
		case string:
			args = append(args, strconv.Quote(arg))
		case *os.File:
			args = append(args, fmt.Sprint(arg.Fd()))
		default:
			args = append(args, fmt.Sprint(arg))
		}
	}

	method := sender.MethodName(r.op)
	return fmt.Sprintf("%v.%v(%v)", sender, method, strings.Join(args, ", "))
}
