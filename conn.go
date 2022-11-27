package wl

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

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

	dir, ok := os.LookupEnv("XDG_RUNTIME_DIR")
	if !ok {
		dir = fmt.Sprintf("/var/run/user/%v", os.Getuid())
	}

	return filepath.Join(dir, v)
}

// Dial opens a connection to the Wayland socket based on the current
// environment. It follows the procedure outlined at
// https://wayland-book.com/protocol-design/wire-protocol.html#transports
func Dial() (*net.UnixConn, error) {
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
		return c.(*net.UnixConn), nil // TODO: Make sure that this works.
	}

	s, err := net.Dial("unix", SocketPath())
	if err != nil {
		return nil, err
	}
	return s.(*net.UnixConn), nil
}
