package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	xdg "deedles.dev/wl/examples/internal/xdg/server"
	wl "deedles.dev/wl/server"
	"deedles.dev/wl/wire"
)

type state struct {
	done  chan struct{}
	close sync.Once

	server *wl.Server
}

func (s *state) init() {
	s.done = make(chan struct{})

	server, err := wl.CreateServer()
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
	s.server = server
	s.server.Handler = s.handleClient
}

func (s *state) stop() {
	s.close.Do(func() { close(s.done) })
}

func (s *state) run(ctx context.Context) {
	log.Printf("display at %q", s.server.Listener.Addr())
	err := s.server.Run(ctx)
	if err != nil {
		log.Fatalf("run server: %v", err)
	}
}

func (s *state) handleClient(ctx context.Context, c *wl.Client) {
	log.Printf("client connected: %p", c)
	defer log.Printf("client disconnected: %p", c)

	cs := clientState{state: (*state)(s), client: c}
	cs.run(ctx)
}

type clientState struct {
	state  *state
	client *wl.Client
	serial uint32

	pingSerial uint32
	pingTime   time.Time
	pongTime   time.Time

	wmBase   *xdg.WmBase
	surfaces []*surface
}

func (cs *clientState) run(ctx context.Context) {
	cs.client.Display().Listener = (*displayListener)(cs)

	ping := time.NewTicker(time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cs.state.done:
			return

		case <-ping.C:
			cs.ping()

		case ev, ok := <-cs.client.Events():
			if !ok {
				return
			}

			err := ev.Flush()
			if err != nil {
				log.Printf("flush: %v", err)
			}
		}
	}
}

func (cs *clientState) serialize() (s uint32) {
	s = cs.serial
	cs.serial++
	return
}

func (cs *clientState) ping() {
	if cs.wmBase == nil {
		return
	}
	if cs.pingTime.After(cs.pongTime) {
		return
	}

	cs.pingSerial = cs.serialize()
	cs.pingTime = time.Now()
	cs.wmBase.Ping(cs.pingSerial)
}

type displayListener clientState

func (cs *displayListener) Sync(cb *wl.Callback) {
	cb.Done((*clientState)(cs).serialize())
}

func (cs *displayListener) GetRegistry(r *wl.Registry) {
	r.Listener = (*registryListener)(cs)
	r.Global(0, wl.CompositorInterface, wl.CompositorVersion)
	r.Global(1, wl.ShmInterface, wl.ShmVersion)
	r.Global(2, xdg.WmBaseInterface, xdg.WmBaseVersion)
}

type registryListener clientState

func (cs *registryListener) Bind(name uint32, id wire.NewID) {
	switch name {
	case 0:
		c := wl.BindCompositor(cs.client, id)
		c.Listener = (*compositorListener)(cs)
	case 1:
		shm := wl.BindShm(cs.client, id)
		shm.Listener = (*shmListener)(cs)
	case 2:
		cs.wmBase = xdg.BindWmBase(cs.client, id)
		cs.wmBase.Listener = (*wmBaseListener)(cs)
	}
}

type compositorListener clientState

func (cs *compositorListener) CreateRegion(r *wl.Region) {
	// TODO
}

func (cs *compositorListener) CreateSurface(s *wl.Surface) {
	cs.surfaces = append(cs.surfaces, &surface{s: s})
}

type shmListener clientState

func (cs *shmListener) CreatePool(pool *wl.ShmPool, file *os.File, size int32) {
	defer file.Close()
	// TODO
}

type wmBaseListener clientState

func (cs *wmBaseListener) Destroy() {}

func (cs *wmBaseListener) CreatePositioner(p *xdg.Positioner) {}

func (cs *wmBaseListener) GetXdgSurface(xs *xdg.Surface, wls *wl.Surface) {
	for _, s := range cs.surfaces {
		if s.s == wls {
			s.role = &xdgRole{s: xs}
			xs.Listener = (*xdgSurfaceListener)(cs)
		}
	}
}

func (cs *wmBaseListener) Pong(serial uint32) {
	if serial != cs.pingSerial {
		return
	}

	cs.pongTime = time.Now()
}

type xdgSurfaceListener clientState

func (cs *xdgSurfaceListener) Destroy() {}

func (cs *xdgSurfaceListener) GetToplevel(tl *xdg.Toplevel) {}

func (cs *xdgSurfaceListener) GetPopup(p *xdg.Popup, parent *xdg.Surface, pos *xdg.Positioner) {}

func (cs *xdgSurfaceListener) SetWindowGeometry(x int32, y int32, width int32, height int32) {}

func (cs *xdgSurfaceListener) AckConfigure(serial uint32) {}

type surface struct {
	s    *wl.Surface
	role any
}

type xdgRole struct {
	s *xdg.Surface
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var s state
	defer s.stop()

	s.init()
	s.run(ctx)
}
