package wl

type Shm struct {
	Format func(uint32)

	obj     shmObject
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

type shmListener struct {
	shm *Shm
}

func (lis shmListener) Format(format uint32) {
	if lis.shm.Format != nil {
		lis.shm.Format(format)
	}
}
