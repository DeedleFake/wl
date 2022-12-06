package main

import (
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sync"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

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
	image image.Image

	once sync.Once
	done chan struct{}

	display    *wl.Display
	registry   *wl.Registry
	shm        *wl.Shm
	compositor *wl.Compositor
	wmBase     *xdg.WmBase
	seat       *wl.Seat
	keyboard   *wl.Keyboard
	pointer    *wl.Pointer

	surface  *wl.Surface
	xsurface *xdg.Surface
	toplevel *xdg.Toplevel
}

func (state *state) init() error {
	state.done = make(chan struct{})

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
	if state.seat == nil {
		return errors.New("no seat found")
	}

	state.surface = state.compositor.CreateSurface()

	state.xsurface = state.wmBase.GetXdgSurface(state.surface)
	state.xsurface.Configure = state.configure

	state.toplevel = state.xsurface.GetToplevel()
	state.toplevel.SetTitle("Example")
	state.toplevel.Close = state.close

	state.keyboard = state.seat.GetKeyboard()

	state.pointer = state.seat.GetPointer()
	state.pointer.Button = state.pointerButton

	state.surface.Commit()

	return nil
}

func (state *state) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-state.done:
			return
		default:
		}

		err := state.display.Flush()
		if err != nil {
			log.Printf("flush: %v", err)
		}
	}
}

func (state *state) close() {
	state.once.Do(func() { close(state.done) })
}

func (state *state) global(name uint32, inter wl.Interface) {
	switch {
	case wl.IsCompositor(inter):
		state.compositor = wl.BindCompositor(state.display, name, inter.Version)
	case wl.IsShm(inter):
		state.shm = wl.BindShm(state.display, name, inter.Version)
	case xdg.IsWmBase(inter):
		state.wmBase = xdg.BindWmBase(state.display, name, inter.Version)
	case wl.IsSeat(inter):
		state.seat = wl.BindSeat(state.display, name, inter.Version)
	}
}

func (state *state) pointerButton(serial, time uint32, button wl.PointerButton, bstate wl.PointerButtonState) {
	switch button {
	case wl.PointerButtonLeft:
		state.toplevel.Move(state.seat, serial)
	}
}

func (state *state) drawFrame() *wl.Buffer {
	bounds := state.image.Bounds().Canon()
	stride := bounds.Dx() * 4
	shmSize := bounds.Dy() * stride

	file, err := shm.Create()
	if err != nil {
		log.Fatalf("create shm: %v", err)
	}
	defer file.Close()
	file.Truncate(int64(shmSize))

	mmap, err := shm.Map(file, shmSize)
	if err != nil {
		log.Fatalf("mmap: %v", err)
	}
	defer mmap.Close()

	pool := state.shm.CreatePool(file, int32(shmSize))
	defer pool.Destroy()
	buf := pool.CreateBuffer(
		0,
		int32(bounds.Dx()),
		int32(bounds.Dy()),
		int32(stride),
		wl.ShmFormatXrgb8888,
	)

	img := shmimage.ARGB8888{
		Pix:    mmap,
		Stride: stride,
		Rect:   image.Rect(0, 0, bounds.Dx(), bounds.Dy()),
	}
	draw.Draw(&img, img.Rect, state.image, bounds.Min, draw.Src)

	return buf
}

func (state *state) configure() {
	buf := state.drawFrame()
	state.surface.Attach(buf, 0, 0)
	state.surface.Commit()
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	return img, err
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: imgview <file>")
		os.Exit(2)
	}

	img, err := loadImage(os.Args[1])
	if err != nil {
		log.Fatalf("load image: %v", err)
	}

	state := state{image: img}
	err = state.init()
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer state.display.Close()

	state.run(ctx)
}
