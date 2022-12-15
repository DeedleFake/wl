package wl

import (
	"context"
	"errors"
	"net"
	"sync"

	"deedles.dev/wl/wire"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -out protocol.go -xml ../protocol/wayland.xml

type Server struct {
	Listener *net.UnixListener

	Handler func(context.Context, *Client)

	err error
}

func CreateServer() (*Server, error) {
	lis, err := wire.Listen()
	if err != nil {
		return nil, err
	}
	return &Server{Listener: lis}, nil
}

func (server *Server) Run(ctx context.Context) (err error) {
	if server.err != nil {
		return server.err
	}

	defer func() {
		if err != nil {
			server.err = err
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		server.Listener.Close()
	}()

	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		c, err := server.Listener.AcceptUnix()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			return err
		}

		wg.Add(1)
		go server.addClient(ctx, &wg, c)
	}
}

func (server *Server) addClient(ctx context.Context, wg *sync.WaitGroup, c *net.UnixConn) {
	defer wg.Done()

	if server.Handler == nil {
		c.Close()
		return
	}

	server.Handler(ctx, newClient(ctx, server, wire.NewConn(c)))
}
