package wl

type Buffer struct {
	Release func()

	I[bufferObject]
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
