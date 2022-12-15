package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	wl "deedles.dev/wl/server"
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
	cs := clientState{state: (*state)(s), client: c}
	cs.init()
}

func (s *serverListener) ClientRemove(c *wl.Client) {}

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
