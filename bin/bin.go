// Package bin contains utilities for dealing with binary representations.
package bin

import (
	"io"
	"unsafe"
)

func Bytes[T ~int32 | ~uint32](v T) [4]byte {
	return *(*[4]byte)(unsafe.Pointer(&v))
}

func Value[T ~int32 | ~uint32](data [4]byte) T {
	return *(*T)(unsafe.Pointer(&data))
}

func Read[T ~int32 | ~uint32](r io.Reader) (T, error) {
	var data [4]byte
	_, err := io.ReadFull(r, data[:])
	if err != nil {
		return 0, err
	}

	return Value[T](data), nil
}

func Write[T ~int32 | ~uint32](w io.Writer, v T) error {
	data := Bytes(v)
	n, err := w.Write(data[:])
	if (err == nil) && (n < len(data)) {
		return io.ErrShortWrite
	}
	return err
}
