package main

import (
	"context"
	"errors"
	"fmt"
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

type state struct {
	display    *wl.Display
	registry   *wl.Registry
	shm        *wl.Shm
	compositor *wl.Compositor
	wmBase     *xdg.WmBase

	surface  *wl.Surface
	xsurface *xdg.Surface
	toplevel *xdg.Toplevel
}

func (state *state) init() error {
	display, err := wl.DialDisplay()
	if err != nil {
		return fmt.Errorf("dial display: %w", err)
	}
	display.Error = func(id, code uint32, msg string) {
		log.Fatalf("display error: id: %v, code: %v, msg: %q", id, code, msg)
	}

	state.display = display
	state.registry = state.display.GetRegistry()
	state.registry.Global = state.global

	err = state.display.RoundTrip()
	if err != nil {
		return fmt.Errorf("round trip: %w", err)
	}

	if state.compositor == nil {
		return errors.New("no compositor found")
	}
	if state.shm == nil {
		return errors.New("no shm found")
	}
	if state.wmBase == nil {
		return errors.New("no wmbase found")
	}

	state.surface = state.compositor.CreateSurface()

	state.xsurface = state.wmBase.GetXdgSurface(state.surface)
	state.xsurface.Configure = state.configure

	state.toplevel = state.xsurface.GetToplevel()
	state.toplevel.SetTitle("Example")

	return nil
}

func (state *state) global(name uint32, inter wl.Interface) {
	switch {
	case wl.IsCompositor(inter):
		state.compositor = wl.BindCompositor(state.display, name)
	case wl.IsShm(inter):
		state.shm = wl.BindShm(state.display, name)
	case xdg.IsWmBase(inter):
		state.wmBase = xdg.BindWmBase(state.display, name)
	}
}

func (state *state) drawFrame() *wl.Buffer {
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
	pool := state.shm.CreatePool(file, ShmSize)
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

	err = unix.Munmap(mmap)
	if err != nil {
		log.Fatalf("munmap: %v", err)
	}

	return buf
}

func (state *state) configure() {
	buf := state.drawFrame()
	state.surface.Attach(buf, 0, 0)
	state.surface.Commit()
}

func (state *state) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := state.display.Flush()
		if err != nil {
			log.Printf("flush: %v", err)
		}
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var state state
	err := state.init()
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer state.display.Close()

	state.run(ctx)
}
