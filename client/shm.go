package wl

import "os"

type Shm struct {
	Format func(ShmFormat)

	I[shmObject]
	display *Display
}

func IsShm(i Interface) bool {
	return i.Is(shmInterface, shmVersion)
}

func BindShm(display *Display, name uint32) *Shm {
	shm := Shm{display: display}
	shm.obj.listener = shmListener{shm: &shm}
	display.AddObject(&shm.obj)

	registry := display.GetRegistry()
	registry.Bind(name, shmInterface, shmVersion, shm.obj.id)

	return &shm
}

func (shm *Shm) CreatePool(file *os.File, size int32) *ShmPool {
	pool := ShmPool{display: shm.display}
	shm.display.AddObject(&pool.obj)
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
