package main

import (
	"context"
	"log"
	"math"
	"os"
	"os/signal"
	"time"

	wl "deedles.dev/wl/client"
	xdg "deedles.dev/xdg/client"
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
	display.Error = func(id, code uint32, msg string) {
		log.Fatalf("display error: id: %v, code: %v, msg: %q", id, code, msg)
	}

	registry := display.GetRegistry()

	var (
		compositor *wl.Compositor
		shm        *wl.Shm
		wmBase     *xdg.WmBase
	)
	registry.Global = func(name uint32, inter wl.Interface) {
		switch {
		case wl.IsCompositor(inter):
			compositor = wl.BindCompositor(display, name)
		case wl.IsShm(inter):
			shm = wl.BindShm(display, name)
		case xdg.IsWmBase(inter):
			wmBase = xdg.BindWmBase(display, name)
		}
	}
	err = display.RoundTrip()
	if err != nil {
		log.Fatalf("round trip: %v", err)
	}

	if compositor == nil {
		log.Fatalln("no compositor found")
	}
	if shm == nil {
		log.Fatalln("no shm found")
	}
	if wmBase == nil {
		log.Fatalln("no wmbase found")
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
	tl := wmBase.GetXdgSurface(surface).GetToplevel()
	tl.SetTitle("Example")

	surface.Attach(buf, 0, 0)
	surface.Damage(0, 0, math.MaxInt32, math.MaxInt32)
	surface.Commit()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := display.Flush()
		if err != nil {
			log.Printf("flush: %v", err)
		}
	}
}
