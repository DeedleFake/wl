package wl

type ShmPool struct {
	I[shmPoolObject]
	display *Display
}

func (pool *ShmPool) CreateBuffer(offset, width, height, stride int32, format ShmFormat) *Buffer {
	buf := Buffer{display: pool.display}
	buf.obj.listener = bufferListener{buf: &buf}
	pool.display.AddObject(&buf.obj)
	pool.display.Enqueue(pool.obj.CreateBuffer(buf.obj.id, offset, width, height, stride, uint32(format)))

	return &buf
}
