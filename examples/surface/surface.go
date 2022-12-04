package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"
	"unsafe"

	wl "deedles.dev/wl/client"
	xdg "deedles.dev/xdg/client"
	"golang.org/x/sys/unix"
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

	surface := compositor.CreateSurface()

	xsurface := wmBase.GetXdgSurface(surface)
	xsurface.Configure = func() {
		const (
			Width   = 640
			Height  = 480
			Stride  = Width * 4
			ShmSize = Height * Stride
		)
		file := CreateShmFile(ShmSize)
		mmap, err := unix.Mmap(int(file.Fd()), 0, ShmSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
		if err != nil {
			log.Fatalf("mmap: %v", err)
		}

		pool := shm.CreatePool(file, ShmSize)
		buf := pool.CreateBuffer(0, Width, Height, Stride, wl.ShmFormatXrgb8888)

		file.Close()

		data := unsafe.Slice((*uint32)(unsafe.Pointer(&mmap[0])), Width*Height)
		for y := 0; y < Height; y++ {
			for x := 0; x < Width; x++ {
				if (x+y/8*8)%16 < 8 {
					data[y*Width+x] = 0xFF666666
					continue
				}
				data[y*Width+x] = 0xFFEEEEEE
			}
		}

		surface.Attach(buf, 0, 0)
		surface.Commit()
	}

	tl := xsurface.GetToplevel()
	tl.SetTitle("Example")

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
