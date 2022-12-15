package wl

import (
	"context"
	"errors"
	"net"
	"sync"

	"deedles.dev/wl/wire"
)

//go:generate go run deedles.dev/wl/cmd/wlgen -out protocol.go -xml ../protocol/wayland.xml

// Server serves the Wayland protocol.
type Server struct {
	// Listener is the Unix socket to listen for incoming connections
	// on.
	Listener *net.UnixListener

	// Handler is called when a new client connects. The lifetime of the
	// client is completely contained to Handler and returning from it
	// will cause the client's connection to be closed.
	Handler func(context.Context, *Client)

	err error
}

// CreateServer creates a default server, setting up a new listener
// for it and setting the server's Listener field.
func CreateServer() (*Server, error) {
	lis, err := wire.Listen()
	if err != nil {
		return nil, err
	}
	return &Server{Listener: lis}, nil
}

// Run runs the server. It does not return until it has completely
// finished and all clients have disconnected. Once this function
// returns, the server is no longer usable and any attempt to run this
// method will immediately return net.ErrClosed.
func (server *Server) Run(ctx context.Context) (err error) {
	if server.err != nil {
		return server.err
	}

	defer func() {
		if err != nil {
			server.err = err
			return
		}
		server.err = net.ErrClosed
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
