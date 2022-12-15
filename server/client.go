package wl

import (
	"errors"
	"io"
	"net"
	"sync"

	"deedles.dev/wl/internal/debug"
	"deedles.dev/wl/internal/objstore"
	"deedles.dev/wl/wire"
)

type Client struct {
	server *Server
	done   chan struct{}
	close  sync.Once
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

	display := NewDisplay(&client)
	display.SetID(1)
	client.store.Add(display)

	go client.listen()

	return &client
}

func (client *Client) listen() {
	defer func() {
		select {
		case <-client.server.done:
		case <-client.done:
		case client.server.queue.Add() <- func() error { client.server.removeClient(client); return nil }:
		}
	}()

	for {
		msg, err := wire.ReadMessage(client.conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}

			select {
			case <-client.server.done:
				return
			case <-client.done:
				return
			case client.server.queue.Add() <- func() error { return err }:
				continue
			}
		}

		select {
		case <-client.server.done:
			return
		case <-client.done:
			return
		case client.server.queue.Add() <- func() error { return client.dispatch(msg) }:
			// TODO: Limit number of queued incoming messages?
		}
	}
}

func (client *Client) dispatch(msg *wire.MessageBuffer) error {
	return client.store.Dispatch(msg)
}

func (client *Client) Add(obj wire.Object) {
	client.store.Add(obj)
}

func (client *Client) Get(id uint32) wire.Object {
	return client.store.Get(id)
}

func (client *Client) Delete(id uint32) {
	client.store.Delete(id)
}

func (client *Client) Enqueue(msg *wire.MessageBuilder) {
	client.server.queue.Add() <- func() error {
		debug.Printf(" -> %v", msg)
		return msg.Build(client.conn)
	}
}

func (client *Client) Display() *Display {
	return client.Get(1).(*Display)
}

func (client *Client) Close() error {
	client.close.Do(func() { close(client.done) })
	return client.conn.Close()
}
