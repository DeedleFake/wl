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

	server, err := wl.ListenAndServe()
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
	s.server = server
	s.server.Listener = (*serverListener)(s)
}

func (s *state) stop() {
	s.close.Do(func() { close(s.done) })
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
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
			err := s.server.Flush()
			if err != nil {
				log.Printf("flush: %v", err)
			}
		}
	}
}

type serverListener state

func (s *serverListener) Client(c *wl.Client) {
	log.Printf("client connected: %v", c)
	cs := clientState{state: (*state)(s), client: c}
	cs.init()
}

func (s *serverListener) ClientRemove(c *wl.Client) {
	log.Printf("client disconnected: %v", c)
}

type clientState struct {
	state  *state
	client *wl.Client
	serial uint32
}

func (cs *clientState) init() {
	cs.client.Display().Listener = (*displayListener)(cs)
}

type displayListener clientState

func (cs *displayListener) Sync(cb *wl.Callback) {
	cb.Done(cs.serial)
	cs.serial++
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
		xdg.BindWmBase(cs.client, id)
	}
}

type compositorListener clientState

func (cs *compositorListener) CreateRegion(r *wl.Region) {
	// TODO
}

func (cs *compositorListener) CreateSurface(s *wl.Surface) {
	// TODO
}

type shmListener clientState

func (cs *shmListener) CreatePool(pool *wl.ShmPool, file *os.File, size int32) {
	defer file.Close()
	// TODO
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var s state
	defer s.stop()

	s.init()
	s.run(ctx)
}
