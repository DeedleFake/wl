package main

import (
	"context"
	"log"
	"math"
	"os"
	"os/signal"
	"time"

	wl "deedles.dev/wl/client"
)

func CreateShmFile(size int64) *os.File {
	path := "/dev/shm/wl-surface-example-" + time.Now().String()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	os.Remove(path)

	err = file.Truncate(size)
	if err != nil {
		panic(err)
	}

	return file
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	display, err := wl.DialDisplay()
	if err != nil {
		log.Fatalf("dial display: %v", err)
	}
	defer display.Close()

	registry := display.GetRegistry()

	var (
		compositor *wl.Compositor
		shm        *wl.Shm
	)
	registry.Global = func(name uint32, inter wl.Interface) {
		switch {
		case wl.IsCompositor(inter):
			compositor = wl.BindCompositor(display, name)
		case wl.IsShm(inter):
			shm = wl.BindShm(display, name)
		}
	}
	err = display.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}

	const (
		Width   = 640
		Height  = 480
		Stride  = Width * 4
		ShmSize = Height * Stride
	)
	file := CreateShmFile(ShmSize)
	//mmap, err := unix.Mmap(int(file.Fd()), 0, ShmSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	//if err != nil {
	//	log.Fatalf("mmap: %v", err)
	//}

	pool := shm.CreatePool(file, ShmSize)
	buf := pool.CreateBuffer(0, Width, Height, Stride, wl.ShmFormatXrgb8888)

	surface := compositor.CreateSurface()
	surface.Attach(buf, 0, 0)
	surface.Damage(0, 0, math.MaxInt32, math.MaxInt32)
	surface.Commit()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err = display.RoundTrip()
		if err != nil {
			log.Fatalf("round trip: %v", err)
		}
	}
}
