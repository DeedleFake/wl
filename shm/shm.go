// Package shm provides helpers for dealing with shared memory.
package shm

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func Create() (*os.File, error) {
	path := "/dev/shm/wl-surface-example-" + time.Now().String()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	return file, os.Remove(path)
}

type Mmap []byte

func Map(file *os.File, size int, prot int) (mmap Mmap, err error) {
	sc, err := file.SyscallConn()
	if err != nil {
		return nil, err
	}

	sc.Control(func(fd uintptr) {
		m, merr := unix.Mmap(int(fd), 0, size, prot, unix.MAP_SHARED)
		mmap, err = Mmap(m), merr
	})

	return mmap, err
}

func (mmap Mmap) Unmap() error {
	return unix.Munmap(mmap)
}
