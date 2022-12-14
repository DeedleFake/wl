package wl

import (
	"deedles.dev/wl/internal/objstore"
	"deedles.dev/wl/wire"
)

type Client struct {
	server *Server
	done   chan struct{}
	conn   *wire.Conn
	store  *objstore.Store
}

func newClient(server *Server, conn *wire.Conn) *Client {
	client := Client{
		server: server,
		done:   make(chan struct{}),
		conn:   conn,
		store:  objstore.New(1 << 24),
	}
	go client.listen()

	return &client
}

func (client *Client) listen() {
	// TODO
}
