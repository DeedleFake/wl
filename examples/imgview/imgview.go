package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	wl "deedles.dev/wl/client"
	"deedles.dev/wl/shm"
	"deedles.dev/wl/shm/shmimage"
	"deedles.dev/wl/wire"
	xdg "deedles.dev/xdg/client"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/colornames"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"golang.org/x/sys/unix"
)

type state struct {
	image image.Image

	close sync.Once
	done  chan struct{}

	state      *wl.State
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

	pointerLoc  image.Point
	barBounds   image.Rectangle
	closeBounds image.Rectangle
	//maxBounds   image.Rectangle
	//max         bool
	minBounds image.Rectangle
}

func (state *state) init() error {
	state.done = make(chan struct{})

	s, err := wl.Dial()
	if err != nil {
		return fmt.Errorf("dial display: %w", err)
	}
	state.state = s

	state.display = s.Display()
	state.display.Listener = (*displayListener)(state)

	state.registry = state.display.GetRegistry()
	state.registry.Listener = (*registryListener)(state)

	err = state.state.RoundTrip()
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
	state.xsurface.Listener = (*xdgSurfaceListener)(state)

	state.toplevel = state.xsurface.GetToplevel()
	state.toplevel.SetTitle("Example")
	state.toplevel.Listener = (*xdgToplevelListener)(state)

	state.keyboard = state.seat.GetKeyboard()

	state.pointer = state.seat.GetPointer()
	state.pointer.Listener = (*pointerListener)(state)

	state.surface.Commit()

	return nil
}

func (state *state) run(ctx context.Context) {
	tick := time.NewTicker(time.Second / 60)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-state.done:
			return
		case <-tick.C:
			err := state.state.Flush()
			if err != nil {
				log.Printf("flush: %v", err)
			}
		}
	}
}

type displayListener state

func (state *displayListener) Error(id, code uint32, msg string) {
	log.Fatalf("display error: id: %v, code: %v, msg: %q", id, code, msg)
}

func (state *displayListener) DeleteId(id uint32) {
	state.state.Delete(id)
}

type registryListener state

func (state *registryListener) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.CompositorInterface:
		state.compositor = wl.BindCompositor(state.state, state.registry, name, version)
	case wl.ShmInterface:
		state.shm = wl.BindShm(state.state, state.registry, name, version)
	case xdg.WmBaseInterface:
		state.wmBase = xdg.BindWmBase(state.state, state.registry, name, version)
		state.wmBase.Listener = (*wmBaseListener)(state)
	case wl.SeatInterface:
		state.seat = wl.BindSeat(state.state, state.registry, name, version)
	}
}

func (state *registryListener) GlobalRemove(name uint32) {}

type wmBaseListener state

func (state *wmBaseListener) Ping(serial uint32) {
	state.wmBase.Pong(serial)
}

type pointerListener state

func (state *pointerListener) Enter(serial uint32, surface *wl.Surface, surfaceX wire.Fixed, surfaceY wire.Fixed) {
}

func (state *pointerListener) Leave(serial uint32, surface *wl.Surface) {}

func (state *pointerListener) Motion(time uint32, x, y wire.Fixed) {
	state.pointerLoc = image.Pt(x.Int(), y.Int())
}

func (state *pointerListener) Button(serial, time uint32, button uint32, bstate wl.PointerButtonState) {
	switch wl.PointerButton(button) {
	case wl.PointerButtonLeft:
		switch {
		case state.pointerLoc.In(state.closeBounds):
			state.close.Do(func() { close(state.done) })
		//case state.pointerLoc.In(state.maxBounds):
		//	state.toplevel.SetMaximized(!state.max)
		//	state.max = !state.max
		case state.pointerLoc.In(state.minBounds):
			state.toplevel.SetMinimized()
		case state.pointerLoc.In(state.barBounds):
			state.toplevel.Move(state.seat, serial)
		}
	}
}

func (state *pointerListener) Axis(time uint32, axis wl.PointerAxis, value wire.Fixed) {}

func (state *pointerListener) Frame() {}

func (state *pointerListener) AxisSource(axisSource wl.PointerAxisSource) {}

func (state *pointerListener) AxisStop(time uint32, axis wl.PointerAxis) {}

func (state *pointerListener) AxisDiscrete(axis wl.PointerAxis, discrete int32) {}

type xdgSurfaceListener state

func (state *xdgSurfaceListener) Configure(serial uint32) {
	state.xsurface.AckConfigure(serial)
}

type xdgToplevelListener state

func (state *xdgToplevelListener) drawFrame(width, height int32) *wl.Buffer {
	const barHeight = 30

	state.barBounds = image.Rect(0, 0, int(width), barHeight)
	imgBounds := image.Rect(0, 0, int(width), int(height))
	if imgBounds.Empty() {
		imgBounds = state.image.Bounds().Canon()
		state.barBounds.Max.X = imgBounds.Max.X
	}
	state.closeBounds = image.Rect(
		state.barBounds.Max.X-(barHeight-8)-4,
		state.barBounds.Min.Y+4,
		state.barBounds.Max.X-4,
		state.barBounds.Max.Y-4,
	)
	//state.maxBounds = state.closeBounds.Sub(image.Pt(barHeight+4, 0))
	state.minBounds = state.closeBounds.Sub(image.Pt(barHeight+4, 0))
	imgBounds = imgBounds.Add(image.Pt(0, barHeight))
	winBounds := state.barBounds.Union(imgBounds)

	stride := winBounds.Dx() * 4
	shmSize := winBounds.Dy() * stride

	file, err := shm.Create()
	if err != nil {
		log.Fatalf("create shm: %v", err)
	}
	defer file.Close()
	file.Truncate(int64(shmSize))

	mmap, err := shm.Map(file, shmSize, unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		log.Fatalf("mmap: %v", err)
	}
	defer mmap.Unmap()

	pool := state.shm.CreatePool(file, int32(shmSize))
	defer pool.Destroy()
	buf := pool.CreateBuffer(
		0,
		int32(winBounds.Dx()),
		int32(winBounds.Dy()),
		int32(stride),
		wl.ShmFormatArgb8888,
	)

	img := shmimage.ARGB8888{
		Pix:    mmap,
		Stride: stride,
		Rect:   winBounds,
	}
	fillRect(&img, state.barBounds, colornames.Dimgray)
	fillRect(&img, state.closeBounds, colornames.Red)
	//fillRect(&img, state.maxBounds, colornames.Limegreen)
	fillRect(&img, state.minBounds, colornames.Yellow)
	draw.ApproxBiLinear.Scale(&img, imgBounds, state.image, state.image.Bounds(), draw.Src, nil)

	return buf
}

func (state *xdgToplevelListener) resize(w, h int32, states []xdg.ToplevelState) {
	buf := state.drawFrame(0, 0)
	state.surface.Attach(buf, 0, 0)
	state.surface.Commit()
}

func (state *xdgToplevelListener) Configure(w, h int32, states []byte) {
	state.resize(0, 0, nil)
}

func (state *xdgToplevelListener) Close() {
	state.close.Do(func() { close(state.done) })
}

func (state *xdgToplevelListener) ConfigureBounds(w, h int32) {}

func (state *xdgToplevelListener) WmCapabilities(capabilities []byte) {}

func fillRect(img draw.Image, r image.Rectangle, c color.Color) {
	r = r.Canon()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
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
	defer state.state.Close()

	state.run(ctx)
}
