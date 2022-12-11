// Package shm provides helpers for dealing with shared memory.
package shm

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// Create returns a file that can be mmapped to share memory with
// another process. It is possible for it to return both a valid file
// and an error.
func Create() (*os.File, error) {
	path := "/dev/shm/wl-surface-example-" + time.Now().String()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	err = os.Remove(path)
	if err != nil {
		err = fmt.Errorf("remove file: %w", err)
	}

	return file, err
}

// Mmap is a []byte that represents a mmapped file.
type Mmap []byte

func mmap(file *os.File, size, prot, flags int) (mmap Mmap, err error) {
	sc, err := file.SyscallConn()
	if err != nil {
		return nil, err
	}

	sc.Control(func(fd uintptr) {
		m, merr := unix.Mmap(int(fd), 0, size, prot, flags)
		mmap, err = Mmap(m), merr
	})

	return mmap, err
}

// MapPrivate maps file into memory at the given size with the given
// prot. It maps it in private mode, meaning that writes to the memory
// will not be reflected in the file.
func MapPrivate(file *os.File, size int, prot int) (Mmap, error) {
	return mmap(file, size, prot, unix.MAP_PRIVATE)
}

// MapShared maps file into memory at the given size with the given
// prot. It maps it in shared mode, meaning that writes to the memory
// will be reflected in the file.
func MapShared(file *os.File, size int, prot int) (Mmap, error) {
	return mmap(file, size, prot, unix.MAP_SHARED)
}

// Unmap unmaps mmap.
func (mmap Mmap) Unmap() error {
	return unix.Munmap(mmap)
}
