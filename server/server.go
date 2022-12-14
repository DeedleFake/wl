package wl

import (
	"errors"
	"net"
	"sync"

	"deedles.dev/wl/internal/cq"
	"deedles.dev/wl/internal/set"
	"deedles.dev/wl/wire"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -out protocol.go -xml ../protocol/wayland.xml

type Server struct {
	done    chan struct{}
	close   sync.Once
	lis     *net.UnixListener
	clients set.Set[*Client]
	queue   *cq.Queue[func() error]
}

func ListenAndServe() (*Server, error) {
	lis, err := wire.Listen()
	if err != nil {
		return nil, err
	}
	return NewServer(lis), nil
}

func NewServer(lis *net.UnixListener) *Server {
	server := Server{
		done:    make(chan struct{}),
		lis:     lis,
		clients: make(set.Set[*Client]),
		queue:   cq.New[func() error](),
	}
	go server.listen()

	return &server
}

func (server *Server) listen() {
	for {
		c, err := server.lis.AcceptUnix()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			select {
			case <-server.done:
				return
			case server.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-server.done:
			return
		case server.queue.Add() <- func() error { server.addClient(c); return nil }:
		}
	}
}

func (server *Server) Close() error {
	server.close.Do(func() { close(server.done) })
	server.queue.Stop()
	return server.lis.Close()
}

func (server *Server) addClient(c *net.UnixConn) {
	server.clients.Add(newClient(server, wire.NewConn(c)))
}

// Flush flushes the event queue, sending all enqueued messages and
// processing all messages that have been received since the last time
// the queue was flushed. It returns all errors encountered.
func (server *Server) Flush() error {
	select {
	case queue := <-server.queue.Get():
		return errors.Join(flushQueue(queue)...)
	default:
		return nil
	}
}

func flushQueue(queue []func() error) (errs []error) {
	for _, ev := range queue {
		err := ev()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
