package wire

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"deedles.dev/wl/internal/set"
	"golang.org/x/sys/unix"
)

func pop[T any, S ~[]T](s S) (v T, ok bool) {
	if len(s) == 0 {
		return v, false
	}

	v = s[0]
	s = s[:len(s)-1]
	copy(s, s[1:cap(s)])
	return v, true
}

func xdgRuntimeDir() string {
	dir, ok := os.LookupEnv("XDG_RUNTIME_DIR")
	if ok {
		return dir
	}
	return fmt.Sprintf("/var/run/user/%v", os.Getuid())
}

// SocketPath determines the path to the Wayland Unix domain socket
// based on the contents of the $WAYLAND_DISPLAY environment variable.
// It does not attempt to determine if the value corresponds to an
// actual socket.
func SocketPath() string {
	v, ok := os.LookupEnv("WAYLAND_DISPLAY")
	if !ok {
		v = "wayland-0"
	}
	if filepath.IsAbs(v) {
		return v
	}

	return filepath.Join(xdgRuntimeDir(), v)
}

// NewSocketPath attempts to generate a valid path for opening a new
// socket to listen on.
func NewSocketPath() (string, error) {
	dir := xdgRuntimeDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	names := make(set.Set[int], len(entries))
	for _, ent := range entries {
		after, ok := strings.CutPrefix(ent.Name(), "wayland-")
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(after, 10, 0)
		if err != nil {
			continue
		}
		names.Add(int(n))
	}

	var num int
	for names.Has(num) {
		num++
	}

	return filepath.Join(dir, fmt.Sprintf("wayland-%v", num)), nil
}

// Conn represents a low-level Wayland connection. It is not generally
// used directly, instead being handled automatically by a State
// implementation.
type Conn struct {
	conn *net.UnixConn
	fds  []int
}

// NewConn creates a new Conn that wraps c. After this is called, use
// the provided Close method to close c instead of calling its own
// Close method.
func NewConn(c *net.UnixConn) *Conn {
	return &Conn{
		conn: c,
	}
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) readFDs(data []byte) error {
	cmsgs, err := unix.ParseSocketControlMessage(data)
	if err != nil {
		return fmt.Errorf("parse socket control messages: %w", err)
	}
	for _, cmsg := range cmsgs {
		fds, err := unix.ParseUnixRights(&cmsg)
		if err != nil {
			if errors.Is(err, unix.EINVAL) {
				continue
			}
			return fmt.Errorf("parse unix control message: %w", err)
		}
		c.fds = append(c.fds, fds...)
	}
	return nil
}

// Dial opens a connection to the Wayland socket based on the current
// environment. It follows the procedure outlined at
// https://wayland-book.com/protocol-design/wire-protocol.html#transports
func Dial() (*Conn, error) {
	if v, ok := os.LookupEnv("WAYLAND_SOCKET"); ok {
		fd, err := strconv.ParseInt(v, 10, 0)
		if err != nil {
			return nil, fmt.Errorf("parse WAYLAND_SOCKET fd: %w", err)
		}
		file := os.NewFile(uintptr(fd), "WAYLAND_SOCKET")
		defer file.Close()

		c, err := net.FileConn(file)
		if err != nil {
			return nil, fmt.Errorf("open WAYLAND_SOCKET connection: %w", err)
		}
		return NewConn(c.(*net.UnixConn)), nil // TODO: Make sure that this works.
	}

	s, err := net.Dial("unix", SocketPath())
	if err != nil {
		return nil, err
	}
	return NewConn(s.(*net.UnixConn)), nil
}
