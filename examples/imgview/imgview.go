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
	xdg "deedles.dev/wl/examples/internal/xdg/client"
	"deedles.dev/wl/pointer"
	"deedles.dev/wl/shm"
	"deedles.dev/wl/wire"
	"deedles.dev/ximage/xcursor"
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

	client     *wl.Client
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
	buffer   *wl.ImageBuffer

	cursorSurface *wl.Surface
	cursorHot     image.Point

	pointerLoc  image.Point
	barBounds   image.Rectangle
	closeBounds image.Rectangle
	//maxBounds   image.Rectangle
	//max         bool
	minBounds image.Rectangle
	highlight *image.Rectangle
}

func (s *state) init() error {
	s.done = make(chan struct{})

	client, err := wl.Dial()
	if err != nil {
		return fmt.Errorf("dial display: %w", err)
	}
	s.client = client

	s.display = client.Display()
	s.display.Listener = (*displayListener)(s)

	s.registry = s.display.GetRegistry()
	s.registry.Listener = (*registryListener)(s)

	err = s.client.RoundTrip()
	if err != nil {
		return fmt.Errorf("round trip: %w", err)
	}

	if s.compositor == nil {
		return errors.New("no compositor found")
	}
	if s.shm == nil {
		return errors.New("no shm found")
	}
	if s.wmBase == nil {
		return errors.New("no wmbase found")
	}
	if s.seat == nil {
		return errors.New("no seat found")
	}

	s.initWindow()
	s.initCursor()

	return nil
}

func (s *state) initWindow() {
	s.surface = s.compositor.CreateSurface()

	s.xsurface = s.wmBase.GetXdgSurface(s.surface)
	s.xsurface.Listener = (*xdgSurfaceListener)(s)

	s.toplevel = s.xsurface.GetToplevel()
	s.toplevel.SetTitle("Example")
	s.toplevel.Listener = (*xdgToplevelListener)(s)

	s.keyboard = s.seat.GetKeyboard()

	s.pointer = s.seat.GetPointer()
	s.pointer.Listener = (*pointerListener)(s)

	s.surface.Commit()

	buffer, err := wl.NewImageBuffer(s.shm, 1, 1)
	if err != nil {
		log.Fatalf("create buffer: %v", err)
	}
	s.buffer = buffer
}

func (s *state) initCursor() {
	theme, err := xcursor.LoadTheme("")
	if err != nil {
		log.Fatalf("load cursor theme: %v", err)
	}

	cursors, ok := theme.Cursors["left_ptr"]
	if !ok {
		log.Fatalf("no left_ptr cursor in theme")
	}
	cimg := cursors.Images[cursors.BestSize(32)][0]
	size := len(cimg.Image.Pix)
	s.cursorHot = cimg.Hot

	file, err := shm.Create()
	if err != nil {
		log.Fatalf("create SHM file: %v", err)
	}
	defer file.Close()
	file.Truncate(int64(size))

	mmap, err := shm.MapShared(file, size, unix.PROT_READ|unix.PROT_WRITE)
	if err != nil {
		log.Fatalf("mmap: %v", err)
	}
	defer mmap.Unmap()

	s.cursorSurface = s.compositor.CreateSurface()
	pool := s.shm.CreatePool(file, int32(size))
	defer pool.Destroy()
	buf := pool.CreateBuffer(
		0,
		int32(cimg.Image.Rect.Dx()),
		int32(cimg.Image.Rect.Dy()),
		int32(cimg.Image.Stride()),
		wl.ShmFormatArgb8888,
	)

	copy(mmap, cimg.Image.Pix)

	s.cursorSurface.Attach(buf, 0, 0)
	s.cursorSurface.Commit()
}

func (s *state) run(ctx context.Context) {
	tick := time.NewTicker(time.Second / 60)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case <-tick.C:
			err := s.client.Flush()
			if err != nil {
				log.Printf("flush: %v", err)
			}
		}
	}
}

func (s *state) render(width, height int32) *wl.Buffer {
	const barHeight = 30

	s.barBounds = image.Rect(0, 0, int(width), barHeight)
	imgBounds := image.Rect(0, 0, int(width), int(height))
	if imgBounds.Empty() {
		imgBounds = s.image.Bounds().Canon()
		s.barBounds.Max.X = imgBounds.Max.X
	}
	s.closeBounds = image.Rect(
		s.barBounds.Max.X-(barHeight-8)-4,
		s.barBounds.Min.Y+4,
		s.barBounds.Max.X-4,
		s.barBounds.Max.Y-4,
	)
	//s.maxBounds = s.closeBounds.Sub(image.Pt(barHeight+4, 0))
	s.minBounds = s.closeBounds.Sub(image.Pt(barHeight+4, 0))
	imgBounds = imgBounds.Add(image.Pt(0, barHeight))
	winBounds := s.barBounds.Union(imgBounds)

	s.buffer.Resize(
		int32(winBounds.Dx()),
		int32(winBounds.Dy()),
	)

	img := s.buffer.Image()
	fillRect(img, s.barBounds, colornames.Dimgray)
	fillRect(img, s.closeBounds, colornames.Red)
	//fillRect(img, s.maxBounds, colornames.Limegreen)
	fillRect(img, s.minBounds, colornames.Yellow)
	if s.highlight != nil {
		addRect(img, *s.highlight, color.RGBA{0x10, 0x10, 0x10, 0xFF})
	}
	draw.ApproxBiLinear.Scale(img, imgBounds, s.image, s.image.Bounds(), draw.Src, nil)

	return s.buffer.Buffer()
}

func (s *state) draw(w, h int32) {
	buf := s.render(0, 0)
	s.surface.Attach(buf, 0, 0)
	s.surface.Commit()
}

type displayListener state

func (s *displayListener) Error(id, code uint32, msg string) {
	log.Fatalf("display error: id: %v, code: %v, msg: %q", id, code, msg)
}

func (s *displayListener) DeleteId(id uint32) {
	s.client.Delete(id)
}

type registryListener state

func (s *registryListener) Global(name uint32, inter string, version uint32) {
	switch inter {
	case wl.CompositorInterface:
		s.compositor = wl.BindCompositor(s.client, s.registry, name, version)
	case wl.ShmInterface:
		s.shm = wl.BindShm(s.client, s.registry, name, version)
	case xdg.WmBaseInterface:
		s.wmBase = xdg.BindWmBase(s.client, s.registry, name, version)
		s.wmBase.Listener = (*wmBaseListener)(s)
	case wl.SeatInterface:
		s.seat = wl.BindSeat(s.client, s.registry, name, version)
	}
}

func (s *registryListener) GlobalRemove(name uint32) {}

type wmBaseListener state

func (s *wmBaseListener) Ping(serial uint32) {
	s.wmBase.Pong(serial)
}

type pointerListener state

func (s *pointerListener) Enter(serial uint32, surface *wl.Surface, surfaceX wire.Fixed, surfaceY wire.Fixed) {
	(*state)(s).pointer.SetCursor(serial, s.cursorSurface, int32(s.cursorHot.X), int32(s.cursorHot.Y))
}

func (s *pointerListener) Leave(serial uint32, surface *wl.Surface) {}

func (s *pointerListener) Motion(time uint32, x, y wire.Fixed) {
	s.pointerLoc = image.Pt(x.Int(), y.Int())

	switch {
	case s.pointerLoc.In(s.closeBounds):
		s.highlight = &s.closeBounds
	//case s.pointerLoc.In(s.maxBounds):
	//	s.highlight = &s.maxBounds
	case s.pointerLoc.In(s.minBounds):
		s.highlight = &s.minBounds
	default:
		s.highlight = nil
	}
	//(*state)(s).draw(0, 0)
}

func (s *pointerListener) Button(serial, time uint32, button uint32, bstate wl.PointerButtonState) {
	switch pointer.Button(button) {
	case pointer.ButtonLeft:
		switch {
		case s.pointerLoc.In(s.closeBounds):
			s.close.Do(func() { close(s.done) })
		//case s.pointerLoc.In(s.maxBounds):
		//	s.toplevel.SetMaximized(!s.max)
		//	s.max = !s.max
		case s.pointerLoc.In(s.minBounds):
			s.toplevel.SetMinimized()
		case s.pointerLoc.In(s.barBounds):
			s.toplevel.Move(s.seat, serial)
		}
	}
}

func (s *pointerListener) Axis(time uint32, axis wl.PointerAxis, value wire.Fixed) {}

func (s *pointerListener) Frame() {}

func (s *pointerListener) AxisSource(axisSource wl.PointerAxisSource) {}

func (s *pointerListener) AxisStop(time uint32, axis wl.PointerAxis) {}

func (s *pointerListener) AxisDiscrete(axis wl.PointerAxis, discrete int32) {}

type xdgSurfaceListener state

func (s *xdgSurfaceListener) Configure(serial uint32) {
	s.xsurface.AckConfigure(serial)

	(*state)(s).draw(0, 0)
}

type xdgToplevelListener state

func (s *xdgToplevelListener) Configure(w, h int32, states []byte) {}

func (s *xdgToplevelListener) Close() {
	s.close.Do(func() { close(s.done) })
}

func (s *xdgToplevelListener) ConfigureBounds(w, h int32) {}

func (s *xdgToplevelListener) WmCapabilities(capabilities []byte) {}

func fillRect(img draw.Image, r image.Rectangle, c color.Color) {
	r = r.Canon()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func addRect(img draw.Image, r image.Rectangle, c color.Color) {
	cr, cg, cb, ca := c.RGBA()

	r = r.Canon()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			or, og, ob, oa := img.At(x, y).RGBA()
			img.Set(x, y, color.RGBA{
				uint8((or + cr) * 0xFF / 0xFFFF),
				uint8((og + cg) * 0xFF / 0xFFFF),
				uint8((ob + cb) * 0xFF / 0xFFFF),
				uint8((oa + ca) * 0xFF / 0xFFFF),
			})
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

	s := state{image: img}
	err = s.init()
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer s.client.Close()

	s.run(ctx)
}
