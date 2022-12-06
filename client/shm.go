package wl

import (
	"os"

	"deedles.dev/wl/wire"
)

type Shm struct {
	Format func(ShmFormat)

	obj     shmObject
	display *Display
}

func IsShm(i Interface) bool {
	return i.Is(shmInterface, shmVersion)
}

func BindShm(display *Display, name, version uint32) *Shm {
	shm := Shm{display: display}
	shm.obj.listener = shmListener{shm: &shm}
	display.AddObject(&shm)

	registry := display.GetRegistry()
	registry.Bind(name, shmInterface, version, shm.obj.id)

	return &shm
}

func (shm *Shm) Object() wire.Object {
	return &shm.obj
}

func (shm *Shm) CreatePool(file *os.File, size int32) *ShmPool {
	pool := ShmPool{display: shm.display}
	shm.display.AddObject(&pool)
	shm.display.Enqueue(shm.obj.CreatePool(pool.obj.id, file, size))

	return &pool
}

type shmListener struct {
	shm *Shm
}

func (lis shmListener) Format(format uint32) {
	if lis.shm.Format != nil {
		lis.shm.Format(ShmFormat(format))
	}
}
