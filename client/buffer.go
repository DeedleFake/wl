package wl

import "deedles.dev/wl/wire"

type Buffer struct {
	Release func()

	obj     bufferObject
	display *Display
}

func (buf *Buffer) Object() wire.Object {
	return &buf.obj
}

type bufferListener struct {
	buf *Buffer
}

func (lis bufferListener) Release() {
	if lis.buf.Release != nil {
		lis.buf.Release()
	}
}
