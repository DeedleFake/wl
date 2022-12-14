package wl

import "deedles.dev/wl/wire"

type Client struct {
	server *Server
	done   chan struct{}
	conn   *wire.Conn
}

func newClient(server *Server, conn *wire.Conn) *Client {
	client := Client{
		server: server,
		done:   make(chan struct{}),
		conn:   conn,
	}
	go client.listen()

	return &client
}

func (client *Client) listen() {
	// TODO
}
