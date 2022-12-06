package wl

import "deedles.dev/wl/wire"

type ShmPool struct {
	obj     shmPoolObject
	display *Display
}

func (pool *ShmPool) Object() wire.Object {
	return &pool.obj
}

func (pool *ShmPool) CreateBuffer(offset, width, height, stride int32, format ShmFormat) *Buffer {
	buf := Buffer{display: pool.display}
	buf.obj.listener = bufferListener{buf: &buf}
	pool.display.AddObject(&buf)
	pool.display.Enqueue(pool.obj.CreateBuffer(buf.obj.id, offset, width, height, stride, uint32(format)))

	return &buf
}

func (pool *ShmPool) Destroy() {
	pool.display.Enqueue(pool.obj.Destroy())
	pool.display.DeleteObject(pool.obj.id)
}
