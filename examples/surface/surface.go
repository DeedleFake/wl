package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"

	wl "deedles.dev/wl/client"
	"deedles.dev/wl/shm"
	"deedles.dev/wl/shm/shmimage"
	xdg "deedles.dev/xdg/client"
)

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

	state.surface.Commit()

	return nil
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

	file, err := shm.Create()
	if err != nil {
		log.Fatalf("create shm: %v", err)
	}
	defer file.Close()
	file.Truncate(ShmSize)

	mmap, err := shm.Map(file, ShmSize)
	if err != nil {
		log.Fatalf("mmap: %v", err)
	}
	defer mmap.Close()

	pool := state.shm.CreatePool(file, ShmSize)
	defer pool.Destroy()
	buf := pool.CreateBuffer(0, Width, Height, Stride, wl.ShmFormatXrgb8888)

	img := shmimage.ARGB8888{
		Pix:    mmap,
		Stride: Stride,
		Rect:   image.Rect(0, 0, Width, Height),
	}
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if (x+y/8*8)%16 < 8 {
				img.Set(x, y, shmimage.ARGB8888Color(0xFF666666))
				continue
			}
			img.Set(x, y, shmimage.ARGB8888Color(0xFFEEEEEE))
		}
	}

	return buf
}

func (state *state) configure() {
	buf := state.drawFrame()
	state.surface.Attach(buf, 0, 0)
	state.surface.Commit()
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
