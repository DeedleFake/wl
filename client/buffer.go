package wl

type Buffer struct {
	Release func()

	id[bufferObject]
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
