package wl

type Buffer struct {
	id[bufferObject]

	Release func()

	display *Display
}

type bufferListener struct {
	buf *Buffer
}

func (lis bufferListener) Release() {
	if lis.buf.Release != nil {
		lis.buf.Release()
	}
}
